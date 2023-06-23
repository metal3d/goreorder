package ordering

import "testing"

func TestParse(t *testing.T) {
	const source = `package main

    const C1 = 1
    const C2 = 2
    
    var V1 = 1
    var V2 = 2
    

    const (
        C3 = 3
        C4 = 4
    )

    var (
        V3 = 3
        V4 = 4
    )

    func main() {
        
    }

    type C struct {}
    func NewC() *C { return &C{} }
    func (c *C) C1() {}
    
    type A struct {}
    type B struct {}

    func (a *A) A1() {}
    func (a *A) A2() {}
    func (a *A) A3() {}

    func (b *B) B1() {}
    func (b *B) B2() {}
    func (b *B) B3() {}


    `
	parsed, err := Parse("test.go", []byte(source))
	if err != nil {
		t.Error(err)
	}
	if parsed == nil {
		t.Error("Expected parsed info")
	}
	if len(parsed.Constants) != 3 { // There is 3  blocs, not 4 consts
		t.Errorf("Expected 4 consts, got %d", len(parsed.Constants))
	}
	if len(parsed.Variables) != 3 { // There is 3  blocs, not 4 vars
		t.Errorf("Expected 4 vars, got %d", len(parsed.Variables))
	}
	if len(parsed.Structs) != 3 {
		t.Errorf("Expected 2 structs, got %d", len(parsed.Structs))
	}
	if len(parsed.Constructors) != 1 {
		t.Errorf("Expected 1 constructor, got %d", len(parsed.Constructors))
	}

	for n, m := range parsed.Methods {
		switch n {
		case "A":
			if len(m) != 3 {
				t.Errorf("Expected 3 methods for A, got %d", len(m))
			}
		case "B":
			if len(m) != 3 {
				t.Errorf("Expected 3 methods for B, got %d", len(m))
			}
		case "C":
			if len(m) != 1 {
				t.Errorf("Expected 1 method for C, got %d", len(m))
			}
		default:
			t.Errorf("Unexpected struct %s", n)
		}
	}

	for n, m := range parsed.Constructors {
		switch n {
		case "C":
			if len(m) != 1 {
				t.Errorf("Expected 1 constructor for C, got %d", len(m))
			}
		default:
			t.Errorf("Unexpected struct %s", n)
		}
	}
}
