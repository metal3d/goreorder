package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	logger "github.com/metal3d/goreorder/log"
	"github.com/metal3d/goreorder/ordering"
)

var (
	log = logger.GetLogger()
)

func main() {
	if err := buildMainCommand().Execute(); err != nil {
		fmt.Println(fmt.Errorf("%w", err))
		os.Exit(1)
	}
}

// ReorderConfig is the configuration for the reorder command
type ReorderConfig struct {
	FormatToolName string   `yaml:"format"`
	Write          bool     `yaml:"write"`
	Verbose        bool     `yaml:"verbose"`
	ReorderTypes   bool     `yaml:"reorder-types"`
	MakeDiff       bool     `yaml:"diff"`
	DefOrder       []string `yaml:"order"`
}

func processFile(fileOrDirectoryName string, input []byte, config *ReorderConfig) error {
	if strings.HasSuffix(fileOrDirectoryName, "_test.go") {
		return fmt.Errorf("Skipping test file: " + fileOrDirectoryName)
	}

	if input != nil && len(input) != 0 {
		// process stdin
		content, err := ordering.ReorderSource(ordering.ReorderConfig{
			Filename:       fileOrDirectoryName,
			FormatCommand:  config.FormatToolName,
			ReorderStructs: config.ReorderTypes,
			Diff:           config.MakeDiff,
			Src:            input,
		})
		if err != nil {
			return fmt.Errorf("Error while reordering source: %v", err)
		}
		fmt.Print(string(content))
		return nil
	}

	stat, err := os.Stat(fileOrDirectoryName)
	if err != nil {
		return fmt.Errorf("Error while getting file stat: %v", err)
	}
	if stat.IsDir() {
		// skip vendor directory
		if strings.HasSuffix(fileOrDirectoryName, "vendor") {
			return fmt.Errorf("Skipping vendor directory: " + fileOrDirectoryName)
		}
		// get all files in directory and process them
		log.Println("Processing directory: " + fileOrDirectoryName)
		return filepath.Walk(fileOrDirectoryName, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("Error while walking directory: %v", err)
			}
			if strings.HasSuffix(path, ".go") {
				processFile(path, nil, config)
			}
			return nil
		})
	}

	log.Println("Processing file: " + fileOrDirectoryName)
	output, err := ordering.ReorderSource(ordering.ReorderConfig{
		Filename:       fileOrDirectoryName,
		FormatCommand:  config.FormatToolName,
		ReorderStructs: config.ReorderTypes,
		Diff:           config.MakeDiff,
		DefOrder:       config.DefOrder,
		Src:            input,
	})
	if err != nil {
		return fmt.Errorf("Error while reordering file: %v", err)
	}
	if config.Write {
		err = ioutil.WriteFile(fileOrDirectoryName, []byte(output), 0644)
		if err != nil {
			return fmt.Errorf("Error while writing to file: %v", err)
		}
	} else {
		fmt.Println(output)
	}
	return nil
}

func reorder(config *ReorderConfig, args ...string) error {

	// is there something in stdin?
	filename := ""
	var input []byte
	var err error
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// read from stdin
		input, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Error while reading stdin: %v", err)
		}
		filename = "stdin.go"
		config.Write = false
		log.Println("Processing stdin, write is set to false")
	} else {
		// read from file or directory
		filename = args[0]
		if filename == "" {
			return fmt.Errorf("Filename is empty")
		}
		_, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("Error while getting file stat: %v", err)
		}
	}

	return processFile(filename, input, config)
}
