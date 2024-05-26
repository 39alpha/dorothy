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
		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if !dorothy.IsInitialized() {
			return fmt.Errorf("not a dorothy repository")
		}

		props := strings.Split(args[0], ".")

		return dorothy.SetConfig(props, args[1])
	}),
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
