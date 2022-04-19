package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metal3d/goreorder/ordering"
)

var version = "master" // changed at compilation time

func main() {
	dirname := "."
	filename := ""
	formatExecutable := "gofmt"
	write := false
	verbose := false
	outputFile := ""
	reorderStructs := false
	showVersion := false

	flag.StringVar(&dirname, "dir", dirname, "directory to scan")
	flag.StringVar(&filename, "file", filename, "file to process, deactivates -dir if set")
	flag.BoolVar(&reorderStructs, "reorder-structs", reorderStructs, "reorder structs by name (default: false)")
	flag.BoolVar(&write, "write", write, "write the output to the file, if not set it will print to stdout (default: false)")
	flag.StringVar(&formatExecutable, "format", "gofmt", "the executable to use to format the output")
	flag.StringVar(&outputFile, "output", filename, "output file (default to the original file, only works with -file)")
	flag.BoolVar(&verbose, "verbose", verbose, "get some informations while processing")
	flag.BoolVar(&showVersion, "version", showVersion, "show version ("+version+")")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// only allow gofmt or goimports
	if formatExecutable != "gofmt" && formatExecutable != "goimports" {
		log.Fatal("Only gofmt or goimports are allowed as format executable")
	}

	// check if the executable exists
	if _, err := exec.LookPath(formatExecutable); err != nil {
		log.Fatal("The executable '" + formatExecutable + "' does not exist")
	}

	// is there something in stdin?
	var input []byte
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// yes, read it
		input, _ = ioutil.ReadAll(os.Stdin)
		filename = "stdin.go"
		dirname = ""
		write = false
	}

	if filename != "" {
		log.Println("PROCESS")
		// do not process test files
		if strings.HasSuffix(filename, "_test.go") {
			log.Fatal("Cannot process test files")
		}

		output, err := ordering.ReorderSource(filename, formatExecutable, reorderStructs, input)
		if err != nil {
			log.Fatal(err)
		}
		if write {
			if verbose {
				fmt.Println("Writing to file: " + outputFile)
			}
			err = ioutil.WriteFile(outputFile, []byte(output), 0644)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Println(output)
		}

	} else {
		// Get all files recursively
		files, err := filepath.Glob(filepath.Join(dirname, "*.go"))
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			if verbose {
				log.Println(file)
			}
			output, err := ordering.ReorderSource(file, formatExecutable, reorderStructs, input)
			if err != nil {
				log.Println(err)
				continue
			}
			if write {
				err = ioutil.WriteFile(file, []byte(output), 0644)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(output)
			}
		}
	}
}
