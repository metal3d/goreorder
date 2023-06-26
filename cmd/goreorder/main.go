package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
		"$ %[1]s completion bash -no-descriptions",
		"$ %[1]s completion zsh",
		"$ %[1]s completion fish",
		"$ %[1]s completion powershell",
	}
)

func main() {
	var (
		formatToolName = "gofmt"
		write          = false
		verbose        = false
		reorderStructs = false
		showVersion    = false
		makeDiff       = false
		help           = false
		defOrder       = ordering.DefaultOrder
	)

	cmd := cobra.Command{
		Use:     "goreorder [flags] [file.go|directory|stdin]",
		Short:   "goreorder reorders the vars, const, types... in a Go source file.",
		Example: fmt.Sprintf(strings.Join(examples, "\n"), filepath.Base(os.Args[0])),
		Long:    fmt.Sprintf(usage, filepath.Base(os.Args[0])),
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

	reoderCommand := &cobra.Command{
		Use:   "reorder [flags] [file.go|directory|stdin]",
		Short: "Reorder vars, consts, stucts/types/interaces, methods/functions and constructors in a Go source file.",
		Run: func(cmd *cobra.Command, args []string) {
			stat, _ := os.Stdin.Stat()
			if len(args) == 0 && (stat.Mode()&os.ModeCharDevice) != 0 {
				cmd.Usage()
				os.Exit(1)
			}

			// validate order flags
			if len(defOrder) > 0 {
				for _, v := range defOrder {
					found := false
					for _, w := range ordering.DefaultOrder {
						if v == w {
							found = true
							break
						}
					}
					if !found {
						log.Fatalf("Invalid order name %v, valid order name are %v", v, ordering.DefaultOrder)
					}
				}
			}

			// only allow gofmt or goimports
			if formatToolName != "gofmt" && formatToolName != "goimports" {
				log.Fatal("Only gofmt or goimports are allowed as format executable")
			}

			// check if the executable exists
			if _, err := exec.LookPath(formatToolName); err != nil {
				log.Fatal("The executable '" + formatToolName + "' does not exist")
			}
			logger.SetVerbose(verbose)
			run(formatToolName, reorderStructs, write, makeDiff, defOrder, args...)
		},
	}

	reoderCommand.Flags().StringVarP(&formatToolName, "format", "f", formatToolName, "Format tool to use (gofmt or goimports)")
	reoderCommand.Flags().BoolVarP(&write, "write", "w", write, "Write result to (source) file instead of stdout")
	reoderCommand.Flags().BoolVarP(&verbose, "verbose", "v", verbose, "Verbose output")
	reoderCommand.Flags().BoolVarP(&reorderStructs, "reorder-types", "r", reorderStructs, "Reordering types in addition to methods")
	reoderCommand.Flags().BoolVarP(&makeDiff, "diff", "d", makeDiff, "Make a diff instead of rewriting the file")
	reoderCommand.Flags().StringSliceVarP(&defOrder, "order", "o", defOrder, "Order of the elements. Omitting elements is allowed, the needed elements will be appended")
	cmd.AddCommand(reoderCommand)

	noDocumentation := false
	bashv1Completion := false
	completionCmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Short:     "Generates completion scripts",
		Example:   fmt.Sprintf(strings.Join(completionExamples, "\n"), filepath.Base(os.Args[0])),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				if bashv1Completion {
					cmd.Root().GenBashCompletion(os.Stdout)
					return
				}
				cmd.Root().GenBashCompletionV2(os.Stdout, !noDocumentation)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				cmd.Usage()
				os.Exit(1)
			}
		},
	}
	completionCmd.Flags().BoolVar(&noDocumentation, "no-documentation", noDocumentation, "Do not include documentation")
	completionCmd.Flags().BoolVar(&bashv1Completion, "bashv1", bashv1Completion, "Use bash version 1 completion")

	cmd.AddCommand(completionCmd)
	cmd.Execute()
}

func run(formatToolName string, reorderStructs, write, diff bool, defOrder []ordering.Order, args ...string) {

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
		write = false
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

	processFile(filename, formatToolName, reorderStructs, input, defOrder, write, diff)
}

func processFile(fileOrDirectoryName string, formatToolName string, reorderStructs bool, input []byte, defOrder []ordering.Order, write, diff bool) {
	if strings.HasSuffix(fileOrDirectoryName, "_test.go") {
		log.Println("Skipping test file: " + fileOrDirectoryName)
		return
	}

	if input != nil && len(input) != 0 {
		// process stdin
		content, err := ordering.ReorderSource(ordering.ReorderConfig{
			Filename:       fileOrDirectoryName,
			FormatCommand:  formatToolName,
			ReorderStructs: reorderStructs,
			Src:            input,
			Diff:           diff,
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
				processFile(path, formatToolName, reorderStructs, input, defOrder, write, diff)
			}
			return nil
		})
		return
	}

	log.Println("Processing file: " + fileOrDirectoryName)
	output, err := ordering.ReorderSource(ordering.ReorderConfig{
		Filename:       fileOrDirectoryName,
		FormatCommand:  formatToolName,
		ReorderStructs: reorderStructs,
		Src:            input,
		Diff:           diff,
		DefOrder:       defOrder,
	})
	if err != nil {
		log.Println("ERR: Ordering error:", err)
		return
	}
	if write {
		err = ioutil.WriteFile(fileOrDirectoryName, []byte(output), 0644)
		if err != nil {
			log.Fatal("ERR: Write to file failed:", err)
		}
	} else {
		fmt.Println(output)
	}
}
