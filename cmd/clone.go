package cmd

import (
	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone remote",
	Short: "clone a remote dataset",
	Args:  cobra.RangeArgs(1, 2),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			return err
		}

		var dest string
		if len(args) == 2 {
			dest = args[1]
		} else {
			dest = ""
		}

		if _, err := core.Clone(args[0], dest, global); err != nil {
			return err
		}

		return nil
	}),
}

func init() {
	cloneCmd.Flags().BoolP("global", "g", false, "initialize the repository to use a global IPFS instance")
	rootCmd.AddCommand(cloneCmd)
}
