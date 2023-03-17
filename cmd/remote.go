package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote uri",
	Short: "set the remote in the configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := core.SetRemote(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(remoteCmd)
}
