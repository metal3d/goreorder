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
	content, err := ReorderSource(filename, "gofmt", true)
	if err != nil {
		t.Error(err)
	}

	// check the content
	if content != expectedSource {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedSource, content)
	}

}
