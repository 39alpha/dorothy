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
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
		if err != nil {
			return err
		}
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			return err
		}

		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if noinherit {
			if err := dorothy.ResetConfig(); err != nil {
				return err
			}
		}
		if configpath != "" {
			if err := dorothy.LoadConfigFile(configpath); err != nil {
				return err
			}
		}

		initialized := dorothy.IsInitialized()
		if err = dorothy.Initialize(global); err != nil {
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
	initCmd.Flags().BoolP("global", "g", false, "initialize the repository to use a global IPFS instance")
	rootCmd.AddCommand(initCmd)
}
