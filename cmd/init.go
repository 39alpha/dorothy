package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize a dataset",
	Run: func(cmd *cobra.Command, args []string) {
		d, initialized, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if err = d.Initialize(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if initialized {
			fmt.Println("Dorothy re-initialized")
		} else {
			fmt.Println("Dorothy initialized")
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
