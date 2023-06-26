package main

import (
	"os"
	"testing"
)

func TestBuildCommand(t *testing.T) {
	cmd := buildCommand()
	if cmd == nil {
		t.Error("buildCommand() should not return nil")
	}
}

func TestFull(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	defer os.Chdir(cwd)
	var example = []byte(`package main
type A struct {}
func (a A) A() {}
var B = 1
func main() {}`)

	tmp, err := os.MkdirTemp("", "goreorder-test")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	if err := os.WriteFile("main.go", []byte(example), 0644); err != nil {
		t.Error(err)
	}

	cmd := buildCommand()
	cmd.SetArgs([]string{"reorder", "--write", "./main.go"})
	if err := cmd.Execute(); err != nil {
		t.Error("Command error", err)
	}

	// check file
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Error("Read file error:", err)
	}
	if content == nil {
		t.Error("file should not be nil")
	}

	if string(content) == string(example) {
		t.Error("file should be changed")
	}
}
