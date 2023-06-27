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

const (
	Const     Order = "const"
	Init      Order = "init"
	Main      Order = "main"
	Var       Order = "var"
	Interface Order = "interface"
	Type      Order = "type"
	Func      Order = "func"
)

// DefaultOrder is the default order of elements.
//
// Note, Init and Main are not in the list. If they are present, the init and main functions
// will be moved.
var DefaultOrder = []Order{Const, Var, Interface, Type, Func}

// Order is the type of order, it's an alias of string.
type Order = string

// ReorderConfig is the configuration for the reorder function.
type ReorderConfig struct {
	Filename       string
	FormatCommand  string
	ReorderStructs bool
	Diff           bool
	Src            interface{}
	DefOrder       []Order
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

// const and vars
func processConst(
	info *ParsedInfo,
	constNames, originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
) []string {
	for i, name := range constNames {
		sourceCode := info.Constants[constNames[i]]
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Constants[name].OpeningLine
		}
		for ln := sourceCode.OpeningLine - 1; ln < sourceCode.ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		source = append(source, sourceCode.SourceCode)
		*removedLines += len(info.Constants)
	}
	return source
}

func processFunctions(
	info *ParsedInfo,
	functionNames, originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
	extactinit, extactmain bool,
) []string {
	for _, name := range functionNames {

		if name == "init" && extactinit {
			continue
		}
		if name == "main" && extactmain {
			continue
		}

		sourceCode := info.Functions[name]
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Functions[name].OpeningLine
		}
		for ln := sourceCode.OpeningLine - 1; ln < sourceCode.ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		source = append(source, "\n"+sourceCode.SourceCode)
		*removedLines += len(info.Functions)
	}
	return source
}

func processExtractedFunction(
	info *ParsedInfo,
	functionNames, originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
	funcname string,
) []string {
	for _, name := range functionNames {

		if funcname != name {
			continue
		}

		sourceCode := info.Functions[name]
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Functions[name].OpeningLine
		}
		for ln := sourceCode.OpeningLine - 1; ln < sourceCode.ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		source = append(source, "\n"+sourceCode.SourceCode)
		*removedLines += len(info.Functions)
	}
	return source
}

func processInterfaces(
	info *ParsedInfo,
	originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
) []string {

	for _, name := range *info.InterfaceNames {
		sourceCode := info.Interfaces[name]
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Interfaces[name].OpeningLine
		}
		for ln := sourceCode.OpeningLine - 1; ln < sourceCode.ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		source = append(source, sourceCode.SourceCode)
		*removedLines += len(info.Interfaces)
	}
	return source
}

func processTypes(
	info *ParsedInfo,
	originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
) []string {

	for _, typename := range *info.TypeNames {
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Types[typename].OpeningLine
		}
		// replace the definitions by "// -- line to remove
		for ln := info.Types[typename].OpeningLine - 1; ln < info.Types[typename].ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		*removedLines += info.Types[typename].ClosingLine - info.Types[typename].OpeningLine
		// add the struct definition to "source"
		source = append(source, info.Types[typename].SourceCode)

		// same for constructors
		for _, constructor := range info.Constructors[typename] {
			for ln := constructor.OpeningLine - 1; ln < constructor.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the constructor to "source"
			source = append(source, "\n"+constructor.SourceCode)
		}
		*removedLines += len(info.Constructors[typename])

		// same for methods
		for _, method := range info.Methods[typename] {
			for ln := method.OpeningLine - 1; ln < method.ClosingLine; ln++ {
				originalContent[ln] = "// -- " + sign
			}
			// add the method to "source"
			source = append(source, "\n"+method.SourceCode)
		}
		*removedLines += len(info.Methods[typename])
	}
	return source
}

func processVars(
	info *ParsedInfo,
	varNames, originalContent, source []string,
	removedLines, lineNumberWhereInject *int,
	sign string,
) []string {
	for i, name := range varNames {
		sourceCode := info.Variables[varNames[i]]
		if *removedLines == 0 {
			*lineNumberWhereInject = info.Variables[name].OpeningLine
		}
		for ln := sourceCode.OpeningLine - 1; ln < sourceCode.ClosingLine; ln++ {
			originalContent[ln] = "// -- " + sign
		}
		source = append(source, sourceCode.SourceCode)
		*removedLines += len(info.Variables)
	}
	return source
}

// ReorderSource reorders the source code in the given filename.
// It will be helped by the formatCommand (gofmt or goimports).
// If gofmt is used, the source code will be formatted with the go/fmt package in memory.
//
// This function calls the Parse() function to extract types, methods, vars, consts and constructors.
// Then it replaces the original source code with a comment containing the
// sha256 of the source. This is made to not lose the original source code "lenght"
// while we reinject the ordered source code. Then, we finally
// remove thses lines from the source code.
func ReorderSource(opt ReorderConfig) (string, error) {

	if opt.DefOrder == nil {
		opt.DefOrder = DefaultOrder
	}
	if len(opt.DefOrder) != len(DefaultOrder) {
		// wich one is missing?
		for _, order := range DefaultOrder {
			found := false
			for _, defOrder := range opt.DefOrder {
				if order == defOrder {
					found = true
					break
				}
			}
			if !found {
				// add it to the end
				opt.DefOrder = append(opt.DefOrder, order)
			}
		}
	}

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

	//if len(info.Types) == 0 {
	//	return string(content), errors.New("No structs found in " + opt.Filename + ", cannot reorder")
	//}

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

	functionNames := make([]string, 0, len(info.Functions))
	for functionName := range info.Functions {
		functionNames = append(functionNames, functionName)
	}
	sort.Strings(functionNames)

	varNames := make([]string, 0, len(info.Variables))
	for varName := range info.Variables {
		varNames = append(varNames, varName)
	}
	sort.Strings(varNames)

	constNames := make([]string, 0, len(info.Constants))
	for constName := range info.Constants {
		constNames = append(constNames, constName)
	}
	sort.Strings(constNames)

	if opt.ReorderStructs {
		info.TypeNames.Sort()
	}

	info.InterfaceNames.Sort()

	// Get the source code signature - we will use this to mark the lines to remove later
	sign := fmt.Sprintf("%x", sha256.Sum256(content))

	// We will work on lines.
	originalContent := strings.Split(string(content), "\n")

	// source is the new source code to inject
	source := []string{}

	lineNumberWhereInject := 0
	removedLines := 0

	extactinit := false
	extractmain := false
	for _, order := range opt.DefOrder {
		if order == Init {
			extactinit = true
		}
		if order == Main {
			extractmain = true
		}
	}

	for _, order := range opt.DefOrder {
		switch order {
		case Const:
			source = processConst(
				info,
				constNames, originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign)
		case Var:
			source = processVars(
				info,
				varNames, originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign)
		case Interface:
			source = processInterfaces(
				info,
				originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign,
			)
		case Type:
			source = processTypes(
				info,
				originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign,
			)
		case Func:
			source = processFunctions(
				info,
				functionNames, originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign,
				extactinit, extractmain,
			)
		case Init:
			source = processExtractedFunction(
				info,
				functionNames, originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign,
				"init",
			)
		case Main:
			source = processExtractedFunction(
				info,
				functionNames, originalContent, source,
				&removedLines, &lineNumberWhereInject,
				sign,
				"main",
			)
		}
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
