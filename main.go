package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// GoType represents a struct, method or constructor. The "SourceCode" field contains the doc comment and source in Go, formated and ready to be injected in the source file.
type GoType struct {
	// Name of the struct, method or constructor
	Name string
	// SourceCode contains the doc comment and source in Go, formated and ready to be injected in the source file.
	SourceCode string
	// OpeningLine is the line number where the struct, method or constructor starts in the source file.
	OpeningLine int
	// ClosingLine is the line number where the struct, method or constructor ends in the source file.
	ClosingLine int
}

// Parse the given file and return the methods, constructors and structs.
func Parse(filename, formatCommand string) (map[string][]*GoType, map[string][]*GoType, map[string]*GoType, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	methods := make(map[string][]*GoType)
	constructors := make(map[string][]*GoType)
	structTypes := make(map[string]*GoType)

	sourceCode, _ := ioutil.ReadFile(filename)
	sourceLines := strings.Split(string(sourceCode), "\n")

	// Itrerate over all the top level declarations in the file to find "struct" declarations

	// Iterate over all the top-level declarations in the file. Only find methods for types, set the type as key and the method as value
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				// Method

				// in "func (T) Method(...) ..." get the type T name
				structName := d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Obj.Name

				// Create a new method
				method := &GoType{
					Name:        d.Name.Name,
					OpeningLine: fset.Position(d.Pos()).Line,
					ClosingLine: fset.Position(d.End()).Line,
				}

				// Get the method source code
				comments := GetMethodComments(d)
				method.SourceCode = strings.Join(comments, "\n") + "\n" + strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
				method.OpeningLine -= len(comments)

				// Add the method to the map
				methods[structName] = append(methods[structName], method)

			}
		// find struct declarations
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if s, ok := spec.(*ast.TypeSpec); ok {
						typeDef := &GoType{
							Name:        s.Name.Name,
							OpeningLine: fset.Position(d.Pos()).Line,
							ClosingLine: fset.Position(d.End()).Line,
						}
						comments := GetTypeComments(d)
						typeDef.SourceCode = strings.Join(comments, "\n") + "\n" + strings.Join(sourceLines[typeDef.OpeningLine-1:typeDef.ClosingLine], "\n")
						typeDef.OpeningLine -= len(comments)

						structTypes[s.Name.Name] = typeDef
					}
				}
			}
		}
	}

	// now that we have found types and methods, we will try to find constructors
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				if d.Type == nil || d.Type.Results == nil || len(d.Type.Results.List) == 0 {
					continue
				}

				// in "func Something(...) x, T" get the type T name and check if it's in "methods" map
				// Get the return types
				for _, r := range d.Type.Results.List {
					if exp, ok := r.Type.(*ast.StarExpr); ok {
						if _, ok := exp.X.(*ast.Ident); !ok {
							continue
						}
						returnType := exp.X.(*ast.Ident).Name
						if _, ok := methods[returnType]; !ok {
							// not a contructor for detected types above, skip
							continue
						}
						// Create a new method
						method := &GoType{
							Name:        d.Name.Name,
							OpeningLine: fset.Position(d.Pos()).Line,
							ClosingLine: fset.Position(d.End()).Line,
						}

						// Get the method source code
						comments := GetMethodComments(d)
						method.SourceCode = strings.Join(comments, "\n") + "\n" + strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
						method.OpeningLine -= len(comments)

						// Add the method to the constructors map
						constructors[returnType] = append(constructors[returnType], method)
					}
				}
			}
		}
	}

	return methods, constructors, structTypes, nil
}

// GetMethodComments returns the comments for the given method.
func GetMethodComments(d *ast.FuncDecl) []string {
	if d == nil || d.Doc == nil || d.Doc.List == nil {
		return []string{}
	}

	comments := []string{}
	for _, comment := range d.Doc.List {
		comments = append(comments, comment.Text)
	}
	return comments
}

// GetTypeComments returns the comments for the given type.
func GetTypeComments(d *ast.GenDecl) []string {
	if d == nil || d.Doc == nil || d.Doc.List == nil {
		return []string{}
	}

	comments := []string{}
	for _, comment := range d.Doc.List {
		comments = append(comments, comment.Text)
	}
	return comments
}

// ReorderSource reorders the source code in the given filename. It will be helped by the formatCommand (gofmt or goimports). The method is to
// use the Parse() function to extract types, methods and constructors. Then we replace the original source code with a comment containing the
// sha256 of the source. This is made to not lose the original source code "lenght" while we reinject the ordered source code. Then, we finally
// remove thses lines from the source code.
func ReorderSource(filename, formatCommand string, reorderStructs bool) (string, error) {
	methods, constructors, structs, err := Parse(filename, formatCommand)

	if err != nil {
		return "", err
	}
	if len(structs) == 0 {
		return "", errors.New("No structs found in " + filename + ", cannot reorder")
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
	// get the content of the file
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return "", err
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
		return "", err
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(output)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(formatCommand, "-w", tmpfile.Name())
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// read the temporary file
	content, err = ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func main() {
	dirname := "."
	filename := ""
	formatExecutable := "gofmt"
	write := false
	verbose := false
	outputFile := ""
	reorderStructs := false

	flag.StringVar(&dirname, "dir", dirname, "directory to scan")
	flag.StringVar(&filename, "file", filename, "file to process, deactivates -dir if set")
	flag.BoolVar(&reorderStructs, "reorder-structs", reorderStructs, "reorder structs by name (default: false)")
	flag.BoolVar(&write, "write", write, "write the output to the file, if not set it will print to stdout (default: false)")
	flag.StringVar(&formatExecutable, "format", "gofmt", "the executable to use to format the output")
	flag.StringVar(&outputFile, "output", filename, "output file (default to the original file, only works with -file)")
	flag.BoolVar(&verbose, "verbose", verbose, "get some informations while processing")
	flag.Parse()

	// only allow gofmt or goimports
	if formatExecutable != "gofmt" && formatExecutable != "goimports" {
		log.Fatal("Only gofmt or goimports are allowed as format executable")
	}

	// check if the executable exists
	if _, err := exec.LookPath(formatExecutable); err != nil {
		log.Fatal("The executable '" + formatExecutable + "' does not exist")
	}

	if filename != "" {
		// do not process test files
		if strings.HasSuffix(filename, "_test.go") {
			log.Fatal("Cannot process test files")
		}

		output, err := ReorderSource(filename, formatExecutable, reorderStructs)
		if err != nil {
			log.Fatal(err)
		}
		if write {
			if verbose {
				fmt.Println("Writing to file: " + outputFile)
			}
			err = ioutil.WriteFile(outputFile, []byte(output), 0644)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Println(output)
		}

	} else {
		// Get all files recursively
		files, err := filepath.Glob(filepath.Join(dirname, "*.go"))
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			if verbose {
				log.Println(file)
			}
			output, err := ReorderSource(file, formatExecutable, reorderStructs)
			if err != nil {
				log.Println(err)
				continue
			}
			if write {
				err = ioutil.WriteFile(file, []byte(output), 0644)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(output)
			}
		}
	}
}
