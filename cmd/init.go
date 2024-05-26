package cmd

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize a dataset",
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}
		initialized := dorothy.IsInitialized()

		if err = dorothy.Initialize(); err != nil {
			return err
		}

		if initialized {
			fmt.Println("Dorothy re-initialized")
		} else {
			fmt.Println("Dorothy initialized")
		}

		return nil
	}),
}

func init() {
	rootCmd.AddCommand(initCmd)
}
