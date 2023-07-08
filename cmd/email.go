package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/cli"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email addr",
	Short: "set the user email in the configuration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.SetUserEmail(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	userCmd.AddCommand(emailCmd)
}
