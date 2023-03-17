package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone remote",
	Short: "clone a remote repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := core.Clone(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
