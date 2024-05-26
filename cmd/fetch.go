package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "fetch the current manifest from the remote",
	Run: func(cmd *cobra.Command, args []string) {
		d, initialized, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		} else if !initialized {
			fmt.Fprintf(os.Stderr, "not a dorothy repository")
			os.Exit(1)
		}

		conflicts, err := d.Fetch()
		if len(conflicts) != 0 {
			fmt.Fprintf(os.Stderr, "conflicts:\n")
			for _, conflict := range conflicts {
				fmt.Fprintf(os.Stderr, "  %s", conflict)
			}
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
			os.Exit(1)
		} else if len(conflicts) != 0 {
			fmt.Fprintf(os.Stderr, "fetch failed with conflicts\n")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

}
