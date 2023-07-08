package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/cli"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit path",
	Short: "commit a dataset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message, err := cmd.Flags().GetString("message")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		nopin, err := cmd.Flags().GetBool("no-pin")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		parents, err := cmd.Flags().GetStringSlice("parents")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		pick, err := cmd.Flags().GetBool("pick")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := cli.CommitData(args[0], message, nopin, parents, pick); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringP("message", "m", "", "commit message")
	commitCmd.Flags().BoolP("no-pin", "n", false, "do not pin the data to your local node")
	commitCmd.Flags().StringSliceP("parents", "p", nil, "parents of this commit")
	commitCmd.Flags().BoolP("pick", "P", false, "interactively choose parents (implied by empty --partents)")
}
