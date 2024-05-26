package cmd

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var editorCmd = &cobra.Command{
	Use:   "editor cmd",
	Short: "set the text editor in the configuration file",
	Args:  cobra.ExactArgs(1),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if !dorothy.IsInitialized() {
			return fmt.Errorf("not a dorothy repository")
		}

		return dorothy.SetEditor(args[0])
	}),
}

func init() {
	configCmd.AddCommand(editorCmd)
}
