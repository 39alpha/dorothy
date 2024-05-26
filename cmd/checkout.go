package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout hash dest",
	Short: "checkout a version to a specific destination",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		d, initialized, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		} else if !initialized {
			fmt.Fprintf(os.Stderr, "not dorothy repository\n")
			os.Exit(1)
		}

		if err := d.Checkout(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
