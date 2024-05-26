package cmd

import (
	"github.com/39alpha/dorothy/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start a Dorothy web server",
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
		if err != nil {
			return err
		}
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		var app *server.Server
		if configpath == "" {
			app, err = server.NewServer()
		} else {
			app, err = server.NewServerFromConfigFile(configpath, noinherit)
		}
		if err != nil {
			return err
		}

		return app.ListenOnPort(port)
	}),
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 4248, "port on which to listen")
}
