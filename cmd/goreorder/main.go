package main

import (
	"bytes"
	"fmt"
	"io"
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
		io.WriteString(defaultErrOutpout, fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}
}

// ReorderConfig is the configuration for the reorder command
type ReorderConfig struct {
	FormatToolName string   `yaml:"format"`
	DefOrder       []string `yaml:"order"`
	Write          bool     `yaml:"write"`
	Verbose        bool     `yaml:"verbose"`
	ReorderTypes   bool     `yaml:"reorder-types"`
	MakeDiff       bool     `yaml:"diff"`
}

func reorder(config *ReorderConfig, args ...string) error {

	// is there something in stdin?
	filename := ""
	var input []byte
	var err error
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// read from stdin
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("error while reading stdin: %w", err)
		}
		filename = "stdin.go"
		config.Write = false
		log.Println("Processing stdin, write is set to false")
	} else {
		// read from file or directory
		filename = args[0]
		if filename == "" {
			return fmt.Errorf("filename is empty")
		}
		_, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("error while getting file stat: %w", err)
		}
	}

	return processFile(filename, input, config)
}

func processFile(fileOrDirectoryName string, input []byte, config *ReorderConfig) error {
	if strings.HasSuffix(fileOrDirectoryName, "_test.go") {
		return fmt.Errorf("Skipping test file: " + fileOrDirectoryName)
	}

	if len(input) != 0 {
		// process stdin
		content, err := ordering.ReorderSource(ordering.ReorderConfig{
			Filename:       fileOrDirectoryName,
			FormatCommand:  config.FormatToolName,
			ReorderStructs: config.ReorderTypes,
			Diff:           config.MakeDiff,
			Src:            input,
		})
		if err != nil {
			return fmt.Errorf("error while reordering source: %w", err)
		}
		fmt.Print(string(content))
		return nil
	}

	stat, err := os.Stat(fileOrDirectoryName)
	if err != nil {
		return fmt.Errorf("error while getting file stat: %w", err)
	}
	if stat.IsDir() {
		// skip vendor directory
		if strings.HasSuffix(fileOrDirectoryName, "vendor") {
			return fmt.Errorf("skipping vendor directory: " + fileOrDirectoryName)
		}
		// get all files in directory and process them
		log.Println("Processing directory: " + fileOrDirectoryName)
		return filepath.Walk(fileOrDirectoryName, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error while walking directory: %w", err)
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
		return fmt.Errorf("error while reordering file: %w", err)
	}
	if config.Write {
		err = os.WriteFile(fileOrDirectoryName, []byte(output), 0644)
		if err != nil {
			return fmt.Errorf("error while writing to file: %w", err)
		}
	} else {
		io.Copy(defaultOutpout, bytes.NewBufferString(output))
	}
	return nil
}
