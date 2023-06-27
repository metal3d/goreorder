package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	logger "github.com/metal3d/goreorder/log"
	"github.com/metal3d/goreorder/ordering"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	usage = `%[1]s reorders the types, methods... in a Go
source file. By default, it will print the result to stdout. To allow %[1]s
to write to the file, use the -write flag.`
)

var (
	version  = "master" // changed at compilation time
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
	defaultOutpout io.Writer = os.Stdout
)

func buildCompletionCommand() *cobra.Command {
	noDocumentation := false
	bashv1Completion := false
	completionCmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Short:     "Generates completion scripts",
		Example:   fmt.Sprintf(strings.Join(completionExamples, "\n"), filepath.Base(os.Args[0])),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("shell type required")
			}
			switch args[0] {
			case "bash":
				if bashv1Completion {
					return cmd.Root().GenBashCompletion(defaultOutpout)
				}
				return cmd.Root().GenBashCompletionV2(defaultOutpout, !noDocumentation)
			case "zsh":
				return cmd.Root().GenZshCompletion(defaultOutpout)
			case "fish":
				return cmd.Root().GenFishCompletion(defaultOutpout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(defaultOutpout)
			default:
				return fmt.Errorf("unsupported shell type %q", args[0])
			}
		},
	}
	completionCmd.Flags().BoolVar(
		&noDocumentation,
		"no-documentation", noDocumentation,
		"Do not include documentation")
	completionCmd.Flags().BoolVar(
		&bashv1Completion,
		"bashv1", bashv1Completion,
		"Use bash version 1 completion")

	return completionCmd
}

func buildMainCommand() *cobra.Command {

	cmd := cobra.Command{
		Use:     "goreorder [flags] [file.go|directory|stdin]",
		Short:   "goreorder reorders the vars, const, types... in a Go source file.",
		Example: fmt.Sprintf(strings.Join(examples, "\n"), filepath.Base(os.Args[0])),
		Long:    fmt.Sprintf(usage, filepath.Base(os.Args[0])),
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeViper(cmd, args...)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("You need to specify a command or an option")
		},
	}

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

func buildPrintConfigCommand(config *ReorderConfig, reorderCommand *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "print-config",
		Short: "Print the configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeViper(reorderCommand)
			bindFlags(reorderCommand, viper.GetViper())
			printConfigFile(config)
			return nil
		},
	}
}

func buildReorderCommand(config *ReorderConfig) *cobra.Command {
	reoderCommand := &cobra.Command{
		Use:   "reorder [flags] [file.go|directory|stdin]",
		Short: "Reorder vars, consts, stucts/types/interaces, methods/functions and constructors in a Go source file.",
		RunE: func(cmd *cobra.Command, args []string) error {

			stat, _ := os.Stdin.Stat()
			if len(args) == 0 && (stat.Mode()&os.ModeCharDevice) != 0 {
				return errors.New("You should provide a file or a directory or stream content to stdin.")
			}

			// validate order flags
			validOrder := config.DefOrder
			validOrder = append(validOrder, []string{"main", "init"}...)
			if len(config.DefOrder) > 0 {
				for _, v := range config.DefOrder {
					found := false
					for _, w := range validOrder {
						if v == w {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("Invalid order name %v, valid order name are %v", v, ordering.DefaultOrder)
					}
				}
			}
			// only allow gofmt or goimports
			if config.FormatToolName != "gofmt" && config.FormatToolName != "goimports" {
				return fmt.Errorf("Only gofmt or goimports are allowed")
			}

			// check if the executable exists
			if _, err := exec.LookPath(config.FormatToolName); err != nil {
				return fmt.Errorf("The executable '" + config.FormatToolName + "' does not exist")
			}
			logger.SetVerbose(config.Verbose)
			return reorder(config, args...)
		},
	}

	reoderCommand.Flags().StringVarP(
		&config.FormatToolName,
		"format", "f", config.FormatToolName,
		"Format tool to use (gofmt or goimports)")
	reoderCommand.Flags().BoolVarP(
		&config.Write,
		"write", "w", config.Write,
		"Write result to (source) file instead of stdout")
	reoderCommand.Flags().BoolVarP(
		&config.Verbose,
		"verbose", "v", config.Verbose,
		"Verbose output")
	reoderCommand.Flags().BoolVarP(
		&config.ReorderTypes,
		"reorder-types", "r", config.ReorderTypes,
		"Reordering types in addition to methods")
	reoderCommand.Flags().BoolVarP(
		&config.MakeDiff,
		"diff", "d", config.MakeDiff,
		"Print diff/patch format instead of rewriting the file")
	reoderCommand.Flags().StringSliceVarP(
		&config.DefOrder,
		"order", "o", config.DefOrder,
		`Order of elements when rewriting. You can omit elements, in which case they will 
be placed in the default order after those you have specified.
There are two specific cases: main and init - if they are not specified in the list, 
then they are considered to be functions and will be ordered as such. If you do specify
them, then they will be positioned in the source code in the place you have specified.
- Allowed values are: main, init, `+strings.Join(ordering.DefaultOrder, ", ")+`
- Default order is: `+strings.Join(ordering.DefaultOrder, ","))
	return reoderCommand
}
