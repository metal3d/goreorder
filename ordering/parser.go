package ordering

import (
	"fmt"
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

// ParsedInfo contains information we need to sort in the source file.
type ParsedInfo struct {
	Functions    map[string]*GoType
	Methods      map[string][]*GoType
	Constructors map[string][]*GoType
	Structs      map[string]*GoType
	Constants    map[string]*GoType
	Variables    map[string]*GoType
	StructNames  *StingList
}

// Parse the given file and return the methods, constructors and structs.
func Parse(filename string, src interface{}) (*ParsedInfo, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var (
		methods      = make(map[string][]*GoType)
		functions    = make(map[string]*GoType)
		constructors = make(map[string][]*GoType)
		structTypes  = make(map[string]*GoType)
		structNames  = &StingList{}
		varTypes     = make(map[string]*GoType)
		constTypes   = make(map[string]*GoType)
		sourceCode   []byte
	)

	if src == nil {
		// error should never happen as Parse() worked
		sourceCode, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	} else {
		sourceCode = src.([]byte)
	}
	sourceLines := strings.Split(string(sourceCode), "\n")

	// Iterate over all the top-level declarations in the file.
	// We're looking for type declarations and function declarations. Not constructors yet.
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			findMethods(d, fset, sourceLines, methods)
		// find struct declarations
		case *ast.GenDecl:
			findStructs(d, fset, sourceLines, structNames, structTypes)
			findGlobalVarsAndConsts(d, fset, sourceLines, varTypes, constTypes)
		}
	}

	// Now that we have found types and methods, we will try to find constructors
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			findConstructors(d, fset, sourceLines, methods, constructors)
		}
	}
	// and now functions
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			findFunctions(d, fset, sourceLines, functions, constructors)
		}
	}

	return &ParsedInfo{
		Functions:    functions,
		Structs:      structTypes,
		StructNames:  structNames,
		Methods:      methods,
		Constructors: constructors,
		Variables:    varTypes,
		Constants:    constTypes,
	}, nil
}

// GetMethodComments returns the comments for the given method.
func GetMethodComments(d *ast.FuncDecl) (comments []string) {
	if d == nil || d.Doc == nil || d.Doc.List == nil {
		return
	}

	for _, comment := range d.Doc.List {
		comments = append(comments, comment.Text)
	}
	return
}

// GetTypeComments returns the comments for the given type.
func GetTypeComments(d *ast.GenDecl) (comments []string) {
	if d == nil || d.Doc == nil || d.Doc.List == nil {
		return
	}

	for _, comment := range d.Doc.List {
		comments = append(comments, comment.Text)
	}
	return
}

func findStructs(d *ast.GenDecl, fset *token.FileSet, sourceLines []string, stuctNames *StingList, structTypes map[string]*GoType) {
	if d.Tok != token.TYPE {
		return
	}
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
			stuctNames.Add(s.Name.Name)
		}
	}
}

func findMethods(d *ast.FuncDecl, fset *token.FileSet, sourceLines []string, methods map[string][]*GoType) {

	if d.Recv == nil {
		return
	}
	// Method
	if d.Recv.List == nil || len(d.Recv.List) == 0 { // not a method
		return
	}
	if d.Recv.List[0].Type == nil { // no receiver type... weird but skip
		return
	}
	if _, ok := d.Recv.List[0].Type.(*ast.StarExpr); !ok { // not a pointer receiver
		return
	}
	if d.Recv.List[0].Type.(*ast.StarExpr).X == nil { // no receiver type... weird but skip
		return
	}
	if _, ok := d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident); !ok { // not named receiver, skip
		return
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

func findConstructors(d *ast.FuncDecl, fset *token.FileSet, sourceLines []string, methods, constructors map[string][]*GoType) {

	if d.Type == nil || d.Type.Results == nil || len(d.Type.Results.List) == 0 { // no return type
		return
	}

	// in "func Something(...) x, T" get the type T name and check if it's in "methods" map
	// Get the return types
	for _, r := range d.Type.Results.List {
		if exp, ok := r.Type.(*ast.StarExpr); ok {
			if _, ok := exp.X.(*ast.Ident); !ok {
				continue
			}
			returnType := exp.X.(*ast.Ident).Name
			// Bug: constructors are not detected if the type is not a method receiver
			//if _, ok := methods[returnType]; !ok {
			//	// not a contructor for detected types above, skip
			//	continue
			//}
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

func findGlobalVarsAndConsts(d *ast.GenDecl, fset *token.FileSet, sourceLines []string, varTypes, constTypes map[string]*GoType) {
	if d.Tok != token.VAR && d.Tok != token.CONST {
		return
	}
	for _, spec := range d.Specs {
		if s, ok := spec.(*ast.ValueSpec); ok {
			for _, name := range s.Names {
				// log the source code for the variable or constant
				typeDef := &GoType{
					Name:        name.Name,
					OpeningLine: fset.Position(d.Pos()).Line,
					ClosingLine: fset.Position(d.End()).Line,
				}
				comments := GetTypeComments(d)
				typeDef.SourceCode = strings.Join(comments, "\n") +
					"\n" +
					strings.Join(sourceLines[typeDef.OpeningLine-1:typeDef.ClosingLine], "\n")
				typeDef.OpeningLine -= len(comments)

				// this time, if const or vars are defined in a parenthesis, the source code is the same for all
				// found var or const. So, what we do is to check if the source code is already in the map, and if
				// so, we skip it.
				// we will use the source code signature as the key for the map
				signature := fmt.Sprintf("%d-%d", typeDef.OpeningLine, typeDef.ClosingLine)
				if _, ok := varTypes[signature]; ok {
					continue
				}

				switch d.Tok {
				case token.CONST:
					constTypes[signature] = typeDef
				case token.VAR:
					varTypes[signature] = typeDef
				}
			}
		}
	}
}

func findFunctions(d *ast.FuncDecl, fset *token.FileSet, sourceLines []string, functions map[string]*GoType, constructors map[string][]*GoType) {
	if d.Recv != nil {
		return // because it's a method
	}
	if d.Name == nil {
		return
	}
	if d.Name.Name == "" {
		return
	}

	if inConstructors(constructors, d.Name.Name) {
		return
	}

	functions[d.Name.Name] = &GoType{
		Name:        d.Name.Name,
		OpeningLine: fset.Position(d.Pos()).Line,
		ClosingLine: fset.Position(d.End()).Line,
	}
	comments := GetMethodComments(d)
	functions[d.Name.Name].SourceCode = strings.Join(comments, "\n") +
		"\n" +
		strings.Join(sourceLines[functions[d.Name.Name].OpeningLine-1:functions[d.Name.Name].ClosingLine], "\n")
	functions[d.Name.Name].OpeningLine -= len(comments)
}

func inConstructors(constructorMap map[string][]*GoType, funcname string) bool {
	for _, constructors := range constructorMap {
		for _, constructor := range constructors {
			if constructor.Name == funcname {
				return true
			}
		}
	}
	return false
}
