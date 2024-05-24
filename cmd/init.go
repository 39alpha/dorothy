package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize a dataset",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		d, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if err = d.InitializeDirectory(ctx, "."); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		fmt.Println("Dorothy initialized")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
