package cmd

import (
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "configure the user",
}

func init() {
	configCmd.AddCommand(userCmd)
}
