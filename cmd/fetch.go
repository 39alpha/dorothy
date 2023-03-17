package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/cli"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "fetch the current manifest from the remote",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.Fetch(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

}
