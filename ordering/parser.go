package ordering

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Parse the given file and return the methods, constructors and structs.
func Parse(filename string, src interface{}) (*ParsedInfo, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var (
		methods        = make(map[string][]*GoType)
		functions      = make(map[string]*GoType)
		constructors   = make(map[string][]*GoType)
		types          = make(map[string]*GoType)
		typeNames      = &StingList{}
		interfaceTypes = make(map[string]*GoType)
		interfaceNames = &StingList{}
		varTypes       = make(map[string]*GoType)
		constTypes     = make(map[string]*GoType)
		sourceCode     []byte
	)

	if src == nil {
		// error should never happen as Parse() worked
		sourceCode, err = os.ReadFile(filename)
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
			findTypes(d, fset, sourceLines, typeNames, types)
			findInterfaces(d, fset, sourceLines, interfaceNames, interfaceTypes)
			findGlobalVarsAndConsts(d, fset, sourceLines, varTypes, constTypes)
		}
	}

	// Now that we have found types and methods, we will try to find constructors
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			findConstructors(d, fset, sourceLines, constructors)
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
		Functions:      functions,
		Types:          types,
		TypeNames:      typeNames,
		Interfaces:     interfaceTypes,
		InterfaceNames: interfaceNames,
		Methods:        methods,
		Constructors:   constructors,
		Variables:      varTypes,
		Constants:      constTypes,
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

func findConstructors(d *ast.FuncDecl, fset *token.FileSet, sourceLines []string, constructors map[string][]*GoType) {

	if d.Type == nil || d.Type.Results == nil || len(d.Type.Results.List) == 0 { // no return type
		return
	}

	// in "func Something(...) x, T" get the type T name and check if it's in "methods" map
	// Get the return types
	returnType := ""
	for _, r := range d.Type.Results.List {
		if exp, ok := r.Type.(*ast.StarExpr); ok {
			if _, ok := exp.X.(*ast.Ident); !ok {
				continue
			}
			returnType = exp.X.(*ast.Ident).Name
		} else if exp, ok := r.Type.(*ast.Ident); ok {
			// same as above, but for non-pointer types
			returnType = exp.Name
			// Create a new method
		}
		if returnType == "" {
			return
		}

		// it the method is already in the constructor map, skip it
		//if _, ok := constructors[returnType]; ok {
		//	continue
		//}

		method := &GoType{
			Name:        d.Name.Name,
			OpeningLine: fset.Position(d.Pos()).Line,
			ClosingLine: fset.Position(d.End()).Line,
		}
		// Get the method source code
		comments := GetMethodComments(d)
		if len(comments) > 0 {
			method.SourceCode = strings.Join(comments, "\n") + "\n"
		}
		method.SourceCode += strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
		method.OpeningLine -= len(comments)
		constructors[returnType] = append(constructors[returnType], method)
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
	if len(comments) > 0 {
		functions[d.Name.Name].SourceCode = strings.Join(comments, "\n") + "\n"
	}
	functions[d.Name.Name].SourceCode += strings.Join(sourceLines[functions[d.Name.Name].OpeningLine-1:functions[d.Name.Name].ClosingLine], "\n")
	functions[d.Name.Name].OpeningLine -= len(comments)
}

func findGlobalVarsAndConsts(d *ast.GenDecl, fset *token.FileSet, sourceLines []string, varTypes, constTypes map[string]*GoType) {
	if d.Tok != token.VAR && d.Tok != token.CONST {
		return
	}
	for _, spec := range d.Specs {
		s, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range s.Names {
			parseConstantAndVars(name, d, fset, sourceLines, varTypes, constTypes)
		}
	}
}

func findInterfaces(d *ast.GenDecl, fset *token.FileSet, sourceLines []string, interfaceNames *StingList, interfaceTypes map[string]*GoType) {
	// finc interfaces
	if d.Tok != token.TYPE {
		return
	}
	for _, spec := range d.Specs {
		if s, ok := spec.(*ast.TypeSpec); ok {
			if _, ok := s.Type.(*ast.InterfaceType); ok {
				interfaceDef := &GoType{
					Name:        s.Name.Name,
					OpeningLine: fset.Position(d.Pos()).Line,
					ClosingLine: fset.Position(d.End()).Line,
				}
				comments := GetTypeComments(d)
				if len(comments) > 0 {
					interfaceDef.SourceCode = strings.Join(comments, "\n") + "\n"
				}
				interfaceDef.SourceCode += strings.Join(sourceLines[interfaceDef.OpeningLine-1:interfaceDef.ClosingLine], "\n")
				interfaceDef.OpeningLine -= len(comments)
				interfaceTypes[s.Name.Name] = interfaceDef
				interfaceNames.Add(s.Name.Name)
			}
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

	recv := d.Recv.List[0].Type
	structName := ""
	// if recv is a pointer, get the type it points to
	if _, ok := recv.(*ast.StarExpr); ok {
		structName = d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
	} else if ident, ok := recv.(*ast.Ident); ok {
		structName = ident.Name
	}
	if structName == "" {
		return
	}
	method := &GoType{
		Name:        d.Name.Name,
		OpeningLine: fset.Position(d.Pos()).Line,
		ClosingLine: fset.Position(d.End()).Line,
	}
	comments := GetMethodComments(d)
	if len(comments) > 0 {
		method.SourceCode = strings.Join(comments, "\n") + "\n"
	}
	method.SourceCode += strings.Join(sourceLines[method.OpeningLine-1:method.ClosingLine], "\n")
	method.OpeningLine -= len(comments)
	methods[structName] = append(methods[structName], method)
}

func findTypes(d *ast.GenDecl, fset *token.FileSet, sourceLines []string, typeNames *StingList, types map[string]*GoType) {
	if d.Tok != token.TYPE {
		return
	}
	for _, spec := range d.Specs {
		if s, ok := spec.(*ast.TypeSpec); ok {
			// return if it's an interface
			if _, ok := s.Type.(*ast.InterfaceType); ok {
				return
			}
			typeDef := &GoType{
				Name:        s.Name.Name,
				OpeningLine: fset.Position(d.Pos()).Line,
				ClosingLine: fset.Position(d.End()).Line,
			}
			comments := GetTypeComments(d)
			if len(comments) > 0 {
				typeDef.SourceCode = strings.Join(comments, "\n") + "\n"
			}
			typeDef.SourceCode += strings.Join(sourceLines[typeDef.OpeningLine-1:typeDef.ClosingLine], "\n")
			typeDef.OpeningLine -= len(comments)
			types[s.Name.Name] = typeDef
			typeNames.Add(s.Name.Name)
		}
	}
}

func parseConstantAndVars(name *ast.Ident, d *ast.GenDecl, fset *token.FileSet, sourceLines []string, varTypes, constTypes map[string]*GoType) {

	// log the source code for the variable or constant
	varDef := &GoType{
		Name:        name.Name,
		OpeningLine: fset.Position(d.Pos()).Line,
		ClosingLine: fset.Position(d.End()).Line,
	}
	comments := GetTypeComments(d)
	if len(comments) > 0 {
		varDef.SourceCode = strings.Join(comments, "\n") + "\n"
	}
	varDef.SourceCode += strings.Join(sourceLines[varDef.OpeningLine-1:varDef.ClosingLine], "\n")
	varDef.OpeningLine -= len(comments)

	// this time, if const or vars are defined in a parenthesis, the source code is the same for all
	// found var or const. So, what we do is to check if the source code is already in the map, and if
	// so, we skip it.
	// we will use the source code signature as the key for the map
	signature := fmt.Sprintf("%d-%d", varDef.OpeningLine, varDef.ClosingLine)
	if _, ok := varTypes[signature]; ok {
		return
	}

	switch d.Tok {
	case token.CONST:
		constTypes[signature] = varDef
	case token.VAR:
		varTypes[signature] = varDef
	}

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
