package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "configure the respository",
}

func init() {
	rootCmd.AddCommand(configCmd)
}
