package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	defaultOutpout = bytes.NewBuffer([]byte{})
	defaultErrOutpout = bytes.NewBuffer([]byte{})
}

func TestBuildCommand(t *testing.T) {
	cmd := buildMainCommand()
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

	cmd := buildMainCommand()
	cmd.SetArgs([]string{"reorder", "--write", "./main.go"})
	if err := cmd.Execute(); err != nil {
		t.Error("command error", err)
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

func TestWithDir(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(currentDir)
	tmpDir, err := os.MkdirTemp("", "goreorder-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)
	files := map[string][]byte{
		"main.go": []byte(`package main
type A struct {}
func (a A) A() {}
`),
		"foo/bar.go": []byte(`package foo
type B struct {}
func (b B) B() {}`),
		"foo/baz.go": []byte(`package foo
type C struct {}
func (c C) C() {}`),
	}

	for file, content := range files {
		dirname := filepath.Dir(file)
		if err := os.MkdirAll(dirname, 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(file, content, 0644); err != nil {
			panic(err)
		}
	}

	// launch command
	cmd := buildMainCommand()
	cmd.SetArgs([]string{"reorder", "--write", "./"})
	cmd.Execute()

	// check files
	for file, content := range files {
		newContent, err := os.ReadFile(file)
		if err != nil {
			t.Error(err)
		}
		if string(newContent) == string(content) {
			t.Error("file should be changed")
		}
	}
}

func TestHelp(t *testing.T) {
	cmd := buildMainCommand()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Error("command error", err)
	}

}

func TestVersion(t *testing.T) {
	// show version
	cmd := buildMainCommand()
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Error("version Command error", err)
	}

}

func TestNoArgs(t *testing.T) {
	cmd := buildMainCommand()
	// no arguments
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Error("an error should occur with no argument", err)
	}
}

func TestCompletionCommands(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		cmd := buildMainCommand()
		cmd.SetArgs([]string{"completion", shell})
		if err := cmd.Execute(); err != nil {
			t.Error("command error", err)
		}
	}

	// try bash with --bashv1
	cmd := buildMainCommand()
	cmd.SetArgs([]string{"completion", "bash", "--bashv1"})
	if err := cmd.Execute(); err != nil {
		t.Error("command error", err)
	}

	// with no shell
	cmd = buildMainCommand()
	cmd.SetArgs([]string{"completion"})
	if err := cmd.Execute(); err == nil {
		t.Error("an error should occur with no shell argument", err)
	}

	// and with a bad shell
	cmd = buildMainCommand()
	cmd.SetArgs([]string{"completion", "badshell"})
	if err := cmd.Execute(); err == nil {
		t.Error("an error should occur with a bad shell argument", err)
	}
}
