package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone remote",
	Short: "clone a remote dataset",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var dest string
		if len(args) == 2 {
			dest = args[1]
		} else {
			dest = ""
		}

		if _, err := core.Clone(ctx, args[0], dest); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
