package cmd

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start a Dorothy web server",
	Run: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		var app *server.Server
		if configpath == "" {
			app, err = server.NewServer()
		} else {
			app, err = server.NewServerFromConfigFile(configpath, noinherit)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to start Dorothy\n  %v\n", err)
			os.Exit(1)
		}

		if err := app.ListenOnPort(port); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 4248, "port on which to listen")
	serveCmd.Flags().StringP("config", "c", "", "path to configuration file")
	serveCmd.Flags().BoolP("noinherit", "n", false, "do not inherit options from system configurations")
}
