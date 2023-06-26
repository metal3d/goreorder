package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	logger "github.com/metal3d/goreorder/log"
	"github.com/metal3d/goreorder/ordering"
	"github.com/spf13/cobra"
)

const (
	usage = `%[1]s reorders the types, methods... in a Go
source file. By default, it will print the result to stdout. To allow %[1]s
to write to the file, use the -write flag.`
)

type ReorderConfig struct {
	FormatToolName string   `yaml:"format"`
	Write          bool     `yaml:"write"`
	Verbose        bool     `yaml:"verbose"`
	ReorderTypes   bool     `yaml:"reorder-types"`
	MakeDiff       bool     `yaml:"diff"`
	DefOrder       []string `yaml:"order"`
}

var (
	version  = "master" // changed at compilation time
	log      = logger.GetLogger()
	examples = []string{
		"$ %[1]s reorder --write --reorder-types --format gofmt file.go",
		"$ %[1]s reorder --diff ./mypackage",
		"$ cat file.go | %[1]s reorder",
	}
	completionExamples = []string{
		"$ %[1]s completion bash",
		"$ %[1]s completion bash -no-documentation",
		"$ %[1]s completion zsh",
		"$ %[1]s completion fish",
		"$ %[1]s completion powershell",
	}
)

func main() {
	if err := buildCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}

func buildCommand() *cobra.Command {

	var (
		help        bool
		showVersion bool
	)

	cmd := cobra.Command{
		Use:     "goreorder [flags] [file.go|directory|stdin]",
		Short:   "goreorder reorders the vars, const, types... in a Go source file.",
		Example: fmt.Sprintf(strings.Join(examples, "\n"), filepath.Base(os.Args[0])),
		Long:    fmt.Sprintf(usage, filepath.Base(os.Args[0])),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeViper(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			if help {
				cmd.Usage()
				os.Exit(0)
			}
			if showVersion {
				fmt.Println(version)
				os.Exit(0)
			}

			stat, _ := os.Stdin.Stat()
			if len(args) == 0 && (stat.Mode()&os.ModeCharDevice) != 0 {
				cmd.Usage()
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&showVersion, "version", "V", showVersion, "Show version")
	cmd.Flags().BoolVarP(&help, "help", "h", help, "Show help")

	config := &ReorderConfig{
		FormatToolName: "gofmt",
		Write:          false,
		Verbose:        false,
		ReorderTypes:   false,
		MakeDiff:       false,
	}
	reorderCommand := buildReorderCommand(config)
	cmd.AddCommand(reorderCommand)

	cmd.AddCommand(buildPrintConfigCommand(config, reorderCommand))
	cmd.AddCommand(buildCompletionCommand())
	return &cmd
}

func run(config *ReorderConfig, args ...string) {

	// is there something in stdin?
	filename := ""
	var input []byte
	var err error
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// read from stdin
		input, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		filename = "stdin.go"
		config.Write = false
		log.Println("Processing stdin, write is set to false")
	} else {
		// read from file or directory
		filename = args[0]
		if filename == "" {
			log.Println("filename is empty")
			os.Exit(1)
		}
		_, err := os.Stat(filename)
		if err != nil {
			log.Fatal(err)
		}
	}

	processFile(filename, input, config)
}

func processFile(fileOrDirectoryName string, input []byte, config *ReorderConfig) {
	if strings.HasSuffix(fileOrDirectoryName, "_test.go") {
		log.Println("Skipping test file: " + fileOrDirectoryName)
		return
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
			log.Fatal(err)
		}
		fmt.Print(string(content))
		return
	}

	stat, err := os.Stat(fileOrDirectoryName)
	if err != nil {
		log.Fatal(err)
		return
	}
	if stat.IsDir() {
		// skip vendor directory
		if strings.HasSuffix(fileOrDirectoryName, "vendor") {
			log.Println("Skipping vendor directory: " + fileOrDirectoryName)
			return
		}
		// get all files in directory and process them
		log.Println("Processing directory: " + fileOrDirectoryName)
		filepath.Walk(fileOrDirectoryName, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)
				return err
			}
			if strings.HasSuffix(path, ".go") {
				processFile(path, nil, config)
			}
			return nil
		})
		return
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
		log.Println("ERR: Ordering error:", err)
		return
	}
	if config.Write {
		err = ioutil.WriteFile(fileOrDirectoryName, []byte(output), 0644)
		if err != nil {
			log.Fatal("ERR: Write to file failed:", err)
		}
	} else {
		fmt.Println(output)
	}
}
