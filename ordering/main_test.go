package ordering

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const exampleSourceCode = `package main

// comment 1
// comment 2
type Foo struct {
	// comment 3
	idfoo int
	// comment 4
	namefoo string
}

func (b *Bar) BadlyBar(){
   print("BadlyBar") 
}

// FooMethod1 comment
func (f *Foo) FooMethod1() {
	print("FooMethod1")
}

// Bar doc
type Bar struct {
	// comment 5
	idbar int
	// comment 6
	namebar string
}

// NewFoo doc
func NewFoo() *Foo {
	return nil
}

// NewBar doc
func NewBar() *Bar {
	return nil
}
`

const expectedSource = `package main

// Bar doc
type Bar struct {
	// comment 5
	idbar int
	// comment 6
	namebar string
}

// NewBar doc
func NewBar() *Bar {
	return nil
}

func (b *Bar) BadlyBar() {
	print("BadlyBar")
}

// comment 1
// comment 2
type Foo struct {
	// comment 3
	idfoo int
	// comment 4
	namefoo string
}

// NewFoo doc
func NewFoo() *Foo {
	return nil
}

// FooMethod1 comment
func (f *Foo) FooMethod1() {
	print("FooMethod1")
}
`

func setup() (string, string) {
	// write exampleSourceCode in a temporary file and return the filename
	dirname, err := os.MkdirTemp("", "goreorder-")
	if err != nil {
		panic(err)
	}
	filename := filepath.Join(dirname, "example.go")
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.WriteString(exampleSourceCode)
	if err != nil {
		panic(err)
	}

	return filename, dirname

}

func teardown(files ...string) {
	// remove the temporary file
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			log.Println("ERR: Cannot remove file:", err)
		}
	}
}

func TestReorder(t *testing.T) {
	filename, tmpdir := setup()
	defer teardown(filename, tmpdir)

	// reorder the file
	content, err := ReorderSource(ReorderConfig{
		Filename:       filename,
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            nil,
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}

	// check the content
	if content != expectedSource {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedSource, content)
	}

}

func TestNoStruct(t *testing.T) {
	const source = `package main
    func main() {
        fmt.Println("nothing")
    }
`
	const expected = `package main

func main() {
	fmt.Println("nothing")
}
`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}
	if content != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expected, content)
	}
}

func TestBadFile(t *testing.T) {
	//_, err := ReorderSource("/tmp/foo.go", "gofmt", true, nil, false)
	_, err := ReorderSource(ReorderConfig{
		Filename:       "/tmp/foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            nil,
		Diff:           false,
	})
	if err == nil {
		t.Error("Expected error")
	}
}

func TestSpecialTypes(t *testing.T) {
	const source = `package main

type bar int
type foo int

func main() {
	fmt.Println("nothing")
}
`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}

	if content != source {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", source, content)
	}
}

func TestMethodForNotFoundStruct(t *testing.T) {
	const source = `package main

import "fmt"

type Foo struct {
	idbar   int
	namebar string
}

func main() {
	fmt.Println("nothing")
}

// method for not found struct
func (f *Bar) FooMethod1() {
	fmt.Println("FooMethod1")
}`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}
	if len(content) == 0 {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", source, content)
	}

}

// Test if orphan comments are not lost and not placed in the weird place.
func TestDealWithOrphanComments(t *testing.T) {
	const source = `package main
// orphan comment 1 here

func main() {
	fmt.Println("nothing")
}

// orphan comment 2 here

type Foo struct {}

func (f *Foo) FooMethod1() {}

// foo comment
func foo() {
}

func (f *Foo) FooMethod2() {}

// orphan comment 3 here

func (f *Foo) FooMethod3() {}

// bar comment
func bar() {
}

func (f *Foo) FooMethod4() {}
`

	const expected = `package main

// orphan comment 1 here

// orphan comment 2 here

type Foo struct{}

func (f *Foo) FooMethod1() {}

func (f *Foo) FooMethod2() {}

func (f *Foo) FooMethod3() {}

func (f *Foo) FooMethod4() {}

// bar comment
func bar() {
}

// foo comment
func foo() {
}

func main() {
	fmt.Println("nothing")
}

// orphan comment 3 here
`

	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}
	if content != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expected, content)
	}
}

func TestDiff(t *testing.T) {
	filename, tmpdir := setup()
	defer teardown(filename, tmpdir)

	// for now, only test that no error is returned
	if _, err := ReorderSource(ReorderConfig{
		Filename:       filename,
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            nil,
		Diff:           true,
	}); err != nil {
		t.Error(err)
	}
}

func TestGlobalVarPlace(t *testing.T) {
	const globalVarSource = `package main

var _ Foo = (*Bar)(nil)

const (
    baz1 = 1
    baz2 = 2
)

var (
    bar2 = 2
    bar1 = 1
)

// comment 1
// comment 2
type Foo struct {
    // comment 3
    idfoo int
}

func (f *Foo) FooMethod1() {
    print("FooMethod1")
}
`
	_, err := ReorderSource(ReorderConfig{
		Filename:       "test.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(globalVarSource),
		Diff:           false,
	})

	if err != nil {
		t.Error(err)
	}
}

func TestNoOrderStructs(t *testing.T) {
	const source = `package main
type grault struct {}

type xyzzy struct {}
type bar struct {}
type qux struct {}

type quux struct {}
type corge struct {}
type garply struct {}
type baz struct {}
type waldo struct {}
type fred struct {}
type plugh struct {}
type foo struct {}
`
	const expected = `package main

type grault struct{}
type xyzzy struct{}
type bar struct{}
type qux struct{}
type quux struct{}
type corge struct{}
type garply struct{}
type baz struct{}
type waldo struct{}
type fred struct{}
type plugh struct{}
type foo struct{}
`

	const orderedSource = `package main

type bar struct{}
type baz struct{}
type corge struct{}
type foo struct{}
type fred struct{}
type garply struct{}
type grault struct{}
type plugh struct{}
type quux struct{}
type qux struct{}
type waldo struct{}
type xyzzy struct{}
`

	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}
	if content != expected {
		t.Errorf("Expected UNORDERED:\n%s\nGot:\n%s\n", expected, content)
	}

	content, err = ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: true,
		Src:            []byte(source),
		Diff:           false,
	})
	if err != nil {
		t.Error(err)
	}
	if content != orderedSource {
		t.Errorf("Expected ORDERED:\n%s\nGot:\n%s\n", orderedSource, content)
	}

}

func TestBadFormatCommand(t *testing.T) {
	const source = `package main

import (
    "os"
    "fmt"
)
type grault struct {}
type xyzzy struct {}
type bar struct {}
`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "wthcommand",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if content != source {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", source, content)
	}
}

func TestDifferentOrder(t *testing.T) {
	const source = `package main

var a = 1
const c = 3
var b = 2
`
	const expectedVC = `package main

var a = 1
var b = 2

const c = 3
`
	const expectedCV = `package main

const c = 3

var a = 1
var b = 2
`

	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
		DefOrder:       []Order{"var", "const"},
	})
	if err != nil {
		t.Error(err)
	}
	if content != expectedVC {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedVC, content)
	}

	content, err = ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
		DefOrder:       []Order{"const", "var"},
	})
	if err != nil {
		t.Error(err)
	}
	if content != expectedCV {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedCV, content)
	}
}

func TestOrderInterfaces(t *testing.T) {
	const source = `package main

type Foo interface {
    FooMethod1()
}

type Bar interface {
    BarMethod1()
}
`
	const expected = `package main

type Bar interface {
	BarMethod1()
}
type Foo interface {
	FooMethod1()
}
`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
	})

	if err != nil {
		t.Error(err)
	}

	if content != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expected, content)
	}
}

func TestOrderInterfacesWithComments(t *testing.T) {
	const source = `package main

// Foo is an interface
type Foo interface {
    FooMethod1()
}

// Bar is an interface
type Bar interface {
    BarMethod1()
}
`
	const expected = `package main

// Bar is an interface
type Bar interface {
	BarMethod1()
}

// Foo is an interface
type Foo interface {
	FooMethod1()
}
`
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
	})

	if err != nil {
		t.Error(err)
	}

	if content != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expected, content)
	}
}

func TestExtractFunctions(t *testing.T) {
	const source = `package main
const A = 1
var B = 2

func A() {
    fmt.Println("A")
}
func main() {
    fmt.Println("main")
}
func Bar() {
    fmt.Println("Bar")
}
func Zoo() {
    fmt.Println("Zoo")
}
func init(){

}
type Foo struct {
    A int
}
`

	funcReg := regexp.MustCompile(`func\s+(.*)\s*\(`)
	content, err := ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
	})

	if err != nil {
		t.Error(err)
	}

	// By default, init and main should be last
	funcs := funcReg.FindAllStringSubmatch(content, -1)
	expectedOrder := []string{"A", "Bar", "Zoo", "init", "main"}
	for i, f := range funcs {
		if f[1] != expectedOrder[i] {
			t.Errorf("Expected %s, got %s", expectedOrder[i], f[1])
		}
	}

	content, err = ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
		DefOrder:       []Order{Const, Var, Main},
	})

	if err != nil {
		t.Error(err)
	}

	// Main shuold be right ager vars, and init should be last
	funcs = funcReg.FindAllStringSubmatch(content, -1)
	expectedOrder = []string{"main", "A", "Bar", "Zoo", "init"}
	for i, f := range funcs {
		if f[1] != expectedOrder[i] {
			t.Errorf("Expected %s, got %s", expectedOrder[i], f[1])
		}
	}

	content, err = ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
		DefOrder:       []Order{Const, Var, Init},
	})

	if err != nil {
		t.Error(err)
	}

	// Init should right after vars, and main should be last
	funcs = funcReg.FindAllStringSubmatch(content, -1)
	expectedOrder = []string{"init", "A", "Bar", "Zoo", "main"}
	for i, f := range funcs {
		if f[1] != expectedOrder[i] {
			t.Errorf("Expected %s, got %s", expectedOrder[i], f[1])
		}
	}

	content, err = ReorderSource(ReorderConfig{
		Filename:       "foo.go",
		FormatCommand:  "gofmt",
		ReorderStructs: false,
		Src:            []byte(source),
		Diff:           false,
		DefOrder:       []Order{Const, Var, Init, Main},
	})

	if err != nil {
		t.Error(err)
	}

	// Main and Init should be right after vars, and the rest should be like in default
	funcs = funcReg.FindAllStringSubmatch(content, -1)
	expectedOrder = []string{"init", "main", "A", "Bar", "Zoo"}
	for i, f := range funcs {
		if f[1] != expectedOrder[i] {
			t.Errorf("Expected %s, got %s", expectedOrder[i], f[1])
		}
	}

}
