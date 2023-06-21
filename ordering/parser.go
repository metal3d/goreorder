package ordering

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
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
