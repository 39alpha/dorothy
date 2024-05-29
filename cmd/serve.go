package cmd

import (
	"os"
	"os/signal"

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
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			return err
		}
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		var app *server.Server
		if configpath == "" {
			app, err = server.NewServer(global)
		} else {
			app, err = server.NewServerFromConfigFile(configpath, noinherit, global)
		}
		if err != nil {
			return err
		}

		c := make(chan error, 1)
		go func(c chan error, app *server.Server, port int) {
			c <- app.ListenOnPort(port)
		}(c, app, port)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		for {
			select {
			case <-sigs:
				app.Shutdown()
				return <-c
			}
		}
	}),
}

func init() {
	serveCmd.Flags().BoolP("global", "g", false, "initialize the repository to use a global IPFS instance")
	serveCmd.Flags().IntP("port", "p", 4248, "port on which to listen")
	rootCmd.AddCommand(serveCmd)
}
