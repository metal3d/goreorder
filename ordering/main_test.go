package ordering

import (
	"os"
	"path/filepath"
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

type Bar struct {
	// comment 5
	idbar int
	// comment 6
	namebar string
}

func NewFoo() *Foo {
	return nil
}

func NewBar() *Bar {
	return nil
}
`

const expectedSource = `package main

type Bar struct {
	// comment 5
	idbar int
	// comment 6
	namebar string
}

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

func NewFoo() *Foo {
	return nil
}

// FooMethod1 comment
func (f *Foo) FooMethod1() {
	print("FooMethod1")
}
`

func setup() string {
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

	return filename

}

func teardown(filename string) {
	// remove the temporary file
	os.Remove(filename)
}

func TestReorder(t *testing.T) {
	filename := setup()
	defer teardown(filename)

	// reorder the file
	content, err := ReorderSource(filename, "gofmt", true, nil)
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
	content, err := ReorderSource(source, "gofmt", true, []byte(source))
	if err == nil {
		t.Error("Expected error for no found struct")
	}
	if content != source {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", source, content)
	}
}

func TestBadFile(t *testing.T) {
	_, err := ReorderSource("/tmp/foo.go", "gofmt", true, nil)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestSpecialTypes(t *testing.T) {
	const source = `package main
    type foo int
    type bar int

    func main() {
        fmt.Println("nothing")
    }
    `
	content, err := ReorderSource(source, "gofmt", true, []byte(source))
	if err == nil {
		t.Error("Expected error")
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
	content, err := ReorderSource("foo.go", "gofmt", true, []byte(source))
	if err != nil {
		t.Error(err)
	}
	if len(content) == 0 {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", source, content)
	}

}
