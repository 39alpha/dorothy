package cmd

import (
	"fmt"
	"strings"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var configSetCmd = &cobra.Command{
	Use:   "set <property> <value>",
	Short: "set configuration properties",
	Args:  cobra.ExactArgs(2),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			return err
		}
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

		if !global && configpath == "" && !dorothy.IsInitialized() {
			return fmt.Errorf("not a dorothy repository")
		}

		props := strings.Split(args[0], ".")

		configpath, err = dorothy.SetConfig(props, args[1], global)
		if configpath != "" {
			fmt.Printf("wrote to %q\n", configpath)
		}

		return err
	}),
}

func init() {
	configSetCmd.PersistentFlags().BoolP("global", "g", false, "set global configuration")
	configCmd.AddCommand(configSetCmd)
}
