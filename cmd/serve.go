package cmd

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start a Dorothy server",
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
