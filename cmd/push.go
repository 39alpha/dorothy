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
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
		if err != nil {
			return err
		}

		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if noinherit {
			if err := dorothy.ResetConfig(); err != nil {
				return err
			}
		}
		if configpath != "" {
			if err := dorothy.LoadConfigFile(configpath); err != nil {
				return err
			}
		}

		if err := dorothy.Setup(); err != nil {
			return err
		}

		conflicts, err := dorothy.Push()
		if len(conflicts) != 0 {
			fmt.Fprintf(os.Stderr, "conflicts:\n")
			for _, conflict := range conflicts {
				fmt.Fprintf(os.Stderr, "  %s", conflict)
			}
		}

		if err != nil {
			return fmt.Errorf("fetch failed with errors: %v\n", err)
		} else if len(conflicts) != 0 {
			return fmt.Errorf("fetch failed with conflicts\n")
		}

		return nil
	}),
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
