package ordering

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type ReorderConfig struct {
	Filename       string
	FormatCommand  string
	ReorderStructs bool
	Diff           bool
	Src            interface{}
}

// ReorderSource reorders the source code in the given filename. It will be helped by the formatCommand (gofmt or goimports). The method is to
// use the Parse() function to extract types, methods and constructors. Then we replace the original source code with a comment containing the
// sha256 of the source. This is made to not lose the original source code "lenght" while we reinject the ordered source code. Then, we finally
// remove thses lines from the source code.
func ReorderSource(opt ReorderConfig) (string, error) {
	// in all cases, we must return the original source code if an error occurs
	// get the content of the file

	var content []byte
	var err error
	if opt.Src == nil || len(opt.Src.([]byte)) == 0 {
		content, err = ioutil.ReadFile(opt.Filename)
		if err != nil {
			return "", err
		}
	} else {
		content = opt.Src.([]byte)
	}

	info, err := Parse(opt.Filename, content)

	if err != nil {
		return string(content), errors.New("Error parsing source: " + err.Error())
	}

	if len(info.Structs) == 0 {
		return string(content), errors.New("No structs found in " + opt.Filename + ", cannot reorder")
	}

	// sort methods by name
	for _, method := range info.Methods {
		sort.Slice(method, func(i, j int) bool {
			return method[i].Name < method[j].Name
		})
	}

	for _, constructor := range info.Constructors {
		sort.Slice(constructor, func(i, j int) bool {
			return constructor[i].Name < constructor[j].Name
		})
	}

	if opt.ReorderStructs {
		info.StructNames.Sort()
	}

	// Get the source code signature - we will use this to mark the lines to remove later
	sign := fmt.Sprintf("%x", sha256.Sum256(content))

	// We will work on lines.
	originalContent := strings.Split(string(content), "\n")

	// source is the new source code to inject
	source := []string{}

	lineNumberWhereInject := 0
	removedLines := 0
	for _, typename := range *info.StructNames {
		if removedLines == 0 {
			lineNumberWhereInject = info.Structs[typename].OpeningLine
		}
		// replace the definitions by "// -- line to remove
		for ln := info.Structs[typename].OpeningLine - 1; ln < info.Structs[typename].ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		removedLines += info.Structs[typename].ClosingLine - info.Structs[typename].OpeningLine
		// add the struct definition to "source"
		source = append(source, "\n\n"+info.Structs[typename].SourceCode)

		// same for constructors
		for _, constructor := range info.Constructors[typename] {
			for ln := constructor.OpeningLine - 1; ln < constructor.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the constructor to "source"
			source = append(source, "\n"+constructor.SourceCode)
		}
		removedLines += len(info.Constructors[typename])

		// same for methods
		for _, method := range info.Methods[typename] {
			for ln := method.OpeningLine - 1; ln < method.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the method to "source"
			source = append(source, "\n"+method.SourceCode)
		}
		removedLines += len(info.Methods[typename])
	}

	// add the "source" at the found lineNumberWhereInject
	originalContent = append(originalContent[:lineNumberWhereInject], append(source, originalContent[lineNumberWhereInject:]...)...)

	// remove the lines that were marked as "// -- line to remove"
	temp := []string{}
	for _, line := range originalContent {
		if line != "// -- "+sign {
			temp = append(temp, line)
		}
	}
	originalContent = temp
	output := strings.Join(originalContent, "\n")

	// write in a temporary file and use "gofmt" to format it
	newcontent := []byte(output)
	switch opt.FormatCommand {
	case "gofmt":
		// format the temporary file
		newcontent, err = format.Source([]byte(output))
		if err != nil {
			return string(content), errors.New("Failed to format source: " + err.Error())
		}
	default:
		if newcontent, err = formatWithCommand(content, output, opt); err != nil {
			return string(content), errors.New("Failed to format source: " + err.Error())
		}
	}

	if opt.Diff {
		return doDiff(content, newcontent, opt.Filename)
	}
	return string(newcontent), nil
}

func formatWithCommand(content []byte, output string, opt ReorderConfig) (newcontent []byte, err error) {
	// we use the format command given by the user
	// on a temporary file we need to create and remove
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return content, errors.New("Failed to create temp file: " + err.Error())
	}
	defer os.Remove(tmpfile.Name())

	// write the temporary file
	if _, err := tmpfile.Write([]byte(output)); err != nil {
		return content, errors.New("Failed to write temp file: " + err.Error())
	}
	tmpfile.Close()

	// format the temporary file
	cmd := exec.Command(opt.FormatCommand, "-w", tmpfile.Name())
	if err := cmd.Run(); err != nil {
		return content, err
	}
	// read the temporary file
	newcontent, err = ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return content, errors.New("Read Temporary File error: " + err.Error())
	}
	return newcontent, nil
}
