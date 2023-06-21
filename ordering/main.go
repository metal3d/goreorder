package ordering

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ReorderSource reorders the source code in the given filename. It will be helped by the formatCommand (gofmt or goimports). The method is to
// use the Parse() function to extract types, methods and constructors. Then we replace the original source code with a comment containing the
// sha256 of the source. This is made to not lose the original source code "lenght" while we reinject the ordered source code. Then, we finally
// remove thses lines from the source code.
func ReorderSource(filename, formatCommand string, reorderStructs bool, src interface{}, diff bool) (string, error) {
	// in all cases, we must return the original source code if an error occurs
	// get the content of the file
	var content []byte
	var err error
	if src == nil || len(src.([]byte)) == 0 {
		content, err = ioutil.ReadFile(filename)
		if err != nil {
			return "", err
		}
	} else {
		content = src.([]byte)
	}

	methods, constructors, structs, err := Parse(filename, formatCommand, content)

	if err != nil {
		return string(content), errors.New("Error parsing source: " + err.Error())
	}

	if len(structs) == 0 {
		return string(content), errors.New("No structs found in " + filename + ", cannot reorder")
	}

	// sort methods by name
	for _, method := range methods {
		sort.Slice(method, func(i, j int) bool {
			return method[i].Name < method[j].Name
		})
	}

	for _, method := range constructors {
		sort.Slice(method, func(i, j int) bool {
			return method[i].Name < method[j].Name
		})
	}

	structNames := make([]string, 0, len(methods))
	for _, s := range structs {
		structNames = append(structNames, s.Name)
	}
	if reorderStructs {
		sort.Strings(structNames)
	}

	// Get the source code signature
	sign := fmt.Sprintf("%x", sha256.Sum256(content))

	// We will work on lines.
	originalContent := strings.Split(string(content), "\n")

	// source is the new source code to inject
	source := []string{}

	lineNumberWhereInject := 0
	removedLines := 0
	for _, typename := range structNames {
		if removedLines == 0 {
			lineNumberWhereInject = structs[typename].OpeningLine
		}
		// replace the definitionsby "// -- line to remove
		for ln := structs[typename].OpeningLine - 1; ln < structs[typename].ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		removedLines += structs[typename].ClosingLine - structs[typename].OpeningLine
		// add the struct definition to "source"
		source = append(source, "\n\n"+structs[typename].SourceCode)

		// same for constructors
		for _, constructor := range constructors[typename] {
			for ln := constructor.OpeningLine - 1; ln < constructor.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the constructor to "source"
			source = append(source, "\n"+constructor.SourceCode)
		}
		removedLines += len(constructors[typename])

		// same for methods
		for _, method := range methods[typename] {
			for ln := method.OpeningLine - 1; ln < method.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the method to "source"
			source = append(source, "\n"+method.SourceCode)
		}
		removedLines += len(methods[typename])
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
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return string(content), errors.New("Failed to create temp file: " + err.Error())
	}
	defer func() {
		os.Remove(tmpfile.Name()) // clean up
		tmpfile.Close()
	}()

	if _, err := tmpfile.Write([]byte(output)); err != nil {
		return string(content), errors.New("Failed to write to temporary file: " + err.Error())
	}

	cmd := exec.Command(formatCommand, "-w", tmpfile.Name())
	if err := cmd.Run(); err != nil {
		return string(content), err
	}

	// read the temporary file
	newcontent, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return string(content), errors.New("Read Temporary File error: " + err.Error())
	}

	if diff {
		// create a and b directories in temporary directory
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return string(content), errors.New("Failed to create temp directory: " + err.Error())
		}
		defer os.RemoveAll(tmpDir) // clean up

		// write original content in a
		dirA := filepath.Join(tmpDir, "a")
		dirB := filepath.Join(tmpDir, "b")

		// get the filepath directory from filename and create it in a and b
		dirA = filepath.Join(dirA, filepath.Dir(filename))
		dirB = filepath.Join(dirB, filepath.Dir(filename))

		// and now, it's the same as before
		os.MkdirAll(dirA, 0755)
		os.MkdirAll(dirB, 0755)

		fba := filepath.Join(dirA, filepath.Base(filename))
		if err := ioutil.WriteFile(fba, content, 0644); err != nil {
			return string(content), errors.New("Failed to write to temporary file: " + err.Error())
		}
		fbb := filepath.Join(dirB, filepath.Base(filename))
		if err := ioutil.WriteFile(fbb, newcontent, 0644); err != nil {
			return string(content), errors.New("Failed to write to temporary file: " + err.Error())
		}
		// run diff -Naur a b
		cmd := exec.Command("diff", "-Naur", dirA, dirB)
		out, err := cmd.CombinedOutput()
		if cmd.ProcessState.ExitCode() <= 1 { // 1 is valid, it means there are differences, 0 means no differences
			// remplace tmp/a/ and tmp/b/ with a/ and b/ in the diff output
			// to make it more readable and easier to apply with patch -p1
			out := strings.ReplaceAll(string(out), tmpDir+"/a/", "a/")
			out = strings.ReplaceAll(out, tmpDir+"/b/", "b/")
			return out, nil
		}
		return string(out), err
	}
	return string(newcontent), nil
}
