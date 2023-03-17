package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "name username",
	Short: "set the user name in the configuration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := core.SetUserName(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	userCmd.AddCommand(nameCmd)
}
