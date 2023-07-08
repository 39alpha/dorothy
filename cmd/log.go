package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/cli"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "display the manifest",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.Log(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
