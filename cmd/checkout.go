package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/cli"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout hash dest",
	Short: "checkout a version to a specific destination",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.Checkout(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
