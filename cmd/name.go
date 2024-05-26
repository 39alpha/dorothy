package cmd

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "name username",
	Short: "set the user name in the configuration file",
	Args:  cobra.ExactArgs(1),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if !dorothy.IsInitialized() {
			return fmt.Errorf("not a dorothy repository")
		}

		return dorothy.SetUserName(args[0])
	}),
}

func init() {
	userCmd.AddCommand(nameCmd)
}
