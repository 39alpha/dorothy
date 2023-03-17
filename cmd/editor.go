package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/cli"
	"github.com/spf13/cobra"
)

var editorCmd = &cobra.Command{
	Use:   "editor cmd",
	Short: "set the text editor in the configuration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.SetEditor(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(editorCmd)
}
