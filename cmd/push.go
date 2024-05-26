package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push versions to the remote",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		d, initialized, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		} else if !initialized {
			fmt.Fprintf(os.Stderr, "not a dorothy repository\n")
			os.Exit(1)
		}

		if err := d.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
