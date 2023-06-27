package ordering

const (
	Const     Order = "const"
	Init      Order = "init"
	Main      Order = "main"
	Var       Order = "var"
	Interface Order = "interface"
	Type      Order = "type"
	Func      Order = "func"
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

// Order is the type of order, it's an alias of string.
type Order = string

// ParsedInfo contains information we need to sort in the source file.
type ParsedInfo struct {
	Functions      map[string]*GoType
	Methods        map[string][]*GoType
	Constructors   map[string][]*GoType
	Types          map[string]*GoType
	Interfaces     map[string]*GoType
	Constants      map[string]*GoType
	Variables      map[string]*GoType
	TypeNames      *StingList
	InterfaceNames *StingList
}

// ReorderConfig is the configuration for the reorder function.
type ReorderConfig struct {
	Filename       string
	FormatCommand  string
	ReorderStructs bool
	Diff           bool
	Src            interface{}
	DefOrder       []Order
}
