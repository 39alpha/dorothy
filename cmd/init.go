package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/cli"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize a dataset",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cli.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Dorothy initialized")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
