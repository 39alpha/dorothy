package cmd

import (
	"fmt"
	"strings"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var configGetCmd = &cobra.Command{
	Use:   "get <property>",
	Short: "get configuration properties",
	Args:  cobra.ExactArgs(1),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
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

		if !dorothy.IsInitialized() {
			return fmt.Errorf("not a dorothy repository")
		}

		props := strings.Split(args[0], ".")
		prop, err := dorothy.GetConfig(props)
		if err != nil {
			return err
		}

		fmt.Println(prop)

		return nil
	}),
}

func init() {
	configCmd.AddCommand(configGetCmd)
}
