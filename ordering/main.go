package ordering

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
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
func Parse(filename, formatCommand string, src interface{}) (map[string][]*GoType, map[string][]*GoType, map[string]*GoType, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	methods := make(map[string][]*GoType)
	constructors := make(map[string][]*GoType)
	structTypes := make(map[string]*GoType)

	var sourceCode []byte
	if src == nil {
		// error should never happen as Parse() worked
		sourceCode, _ = ioutil.ReadFile(filename)
	} else {
		sourceCode = src.([]byte)
	}
	sourceLines := strings.Split(string(sourceCode), "\n")

	// Iterate over all the top-level declarations in the file. Only find methods for types, set the type as key and the method as value
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				// Method

				if d.Recv.List == nil || len(d.Recv.List) == 0 {
					continue
				}
				if d.Recv.List[0].Type == nil {
					continue
				}
				if _, ok := d.Recv.List[0].Type.(*ast.StarExpr); !ok {
					continue
				}
				if d.Recv.List[0].Type.(*ast.StarExpr).X == nil {
					continue
				}
				if _, ok := d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident); !ok {
					continue
				}

				// in "func (T) Method(...) ..." get the type T name
				structName := d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

				// Create a new method
				method := &GoType{
					Name:        d.Name.Name,
					OpeningLine: fset.Position(d.Pos()).Line,
					ClosingLine: fset.Position(d.End()).Line,
				}

				// Get the method source code
				comments := GetMethodComments(d)
				method.SourceCode = strings.Join(comments, "\n") +
					"\n" +
					strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
				method.OpeningLine -= len(comments)

				// Add the method to the map
				methods[structName] = append(methods[structName], method)

			}
		// find struct declarations
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if s, ok := spec.(*ast.TypeSpec); ok {
						// is it a struct?
						if _, ok := s.Type.(*ast.StructType); !ok {
							// no... skip
							continue
						}
						typeDef := &GoType{
							Name:        s.Name.Name,
							OpeningLine: fset.Position(d.Pos()).Line,
							ClosingLine: fset.Position(d.End()).Line,
						}
						comments := GetTypeComments(d)
						typeDef.SourceCode = strings.Join(comments, "\n") +
							"\n" +
							strings.Join(sourceLines[typeDef.OpeningLine-1:typeDef.ClosingLine], "\n")
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
						method.SourceCode = strings.Join(comments, "\n") +
							"\n" +
							strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
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
