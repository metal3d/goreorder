package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	logger "github.com/metal3d/goreorder/log"
	"github.com/metal3d/goreorder/ordering"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func buildCompletionCommand() *cobra.Command {
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
			if len(config.DefOrder) > 0 {
				for _, v := range config.DefOrder {
					found := false
					for _, w := range ordering.DefaultOrder {
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
			run(config, args...)
			return nil
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
		"Make a diff instead of rewriting the file")
	reoderCommand.Flags().StringSliceVarP(
		&config.DefOrder,
		"order", "o", config.DefOrder,
		"Order of the elements. Omitting elements is allowed, the needed elements will be appended")
	return reoderCommand

}
