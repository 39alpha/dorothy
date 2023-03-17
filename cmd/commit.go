package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit path",
	Short: "commit a dataset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nopin, err := cmd.Flags().GetBool("no-pin")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		message, err := cmd.Flags().GetString("message")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := core.CommitData(args[0], message, nopin); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringP("message", "m", "", "commit message")
	commitCmd.Flags().BoolP("no-pin", "n", false, "do not pin the data to your local node")
}
