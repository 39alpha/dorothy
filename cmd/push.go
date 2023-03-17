package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push versions to the remote",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := core.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
