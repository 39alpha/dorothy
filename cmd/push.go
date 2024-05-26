package cmd

import (
	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push versions to the remote",
	Args:  cobra.ExactArgs(0),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if err := dorothy.Setup(); err != nil {
			return err
		}

		return dorothy.Push()
	}),
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
