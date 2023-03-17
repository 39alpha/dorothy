package cmd

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start a Dorthy server",
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
