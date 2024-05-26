package cmd

import (
	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout hash dest",
	Short: "checkout a version to a specific destination",
	Args:  cobra.ExactArgs(2),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if err := dorothy.Setup(); err != nil {
			return err
		}

		return dorothy.Checkout(args[0], args[1])
	}),
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
