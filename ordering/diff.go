package ordering

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// doDiff returns the unified diff between content and newcontent
func doDiff(content, newcontent []byte, filename string) (string, error) {
	// this function creates a temporary directory, and write the content and newcontent in two directories
	// named "a" and "b". Then, it runs "diff -Naur a b" and returns the output.
	// To avoid problems with the patch command, the output is modified to replace
	// "tmp/a/" and "tmp/b/" with "a/" and "b/".

	// create a and b directories in temporary directory
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return string(content), errors.New("Failed to create temp directory: " + err.Error())
	}
	defer os.RemoveAll(tmpDir) // clean up

	// write original content in a/ and newcontent in b/
	filedir := filepath.Dir(filename)
	filebase := filepath.Base(filename)
	dirA := filepath.Join(tmpDir, "a", filedir)
	dirB := filepath.Join(tmpDir, "b", filedir)

	// and now, it's the same as before
	os.MkdirAll(dirA, 0755)
	os.MkdirAll(dirB, 0755)

	fba := filepath.Join(dirA, filebase)
	if err := ioutil.WriteFile(fba, content, 0644); err != nil {
		return string(content), errors.New("Failed to write to temporary file: " + err.Error())
	}
	fbb := filepath.Join(dirB, filebase)
	if err := ioutil.WriteFile(fbb, newcontent, 0644); err != nil {
		return string(content), errors.New("Failed to write to temporary file: " + err.Error())
	}
	// run diff -Naur a b
	cmd := exec.Command("diff", "-Naur", dirA, dirB)
	out, err := cmd.CombinedOutput()
	if cmd.ProcessState.ExitCode() <= 1 { // 1 is valid, it means there are differences, 0 means no differences
		// remplace tmp/a/ and tmp/b/ with a/ and b/ in the diff output
		// to make it more readable and easier to apply with patch -p1
		out := strings.ReplaceAll(string(out), tmpDir+"/a/", "a/")
		out = strings.ReplaceAll(out, tmpDir+"/b/", "b/")
		return out, nil
	}
	return string(out), err
}
