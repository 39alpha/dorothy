package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type CommandWithError func(*cobra.Command, []string) error

type Command func(cmd *cobra.Command, args []string)

func HandleErrors(f CommandWithError) Command {
	return func(cmd *cobra.Command, args []string) {
		if err := f(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "dorothy",
	Short: "A stab at data management",
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to configuration file")
	rootCmd.PersistentFlags().BoolP("noinherit", "n", false, "do not inherit options from system configurations")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
