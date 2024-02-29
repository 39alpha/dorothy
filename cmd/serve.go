package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/39alpha/dorothy/cli"
	"github.com/spf13/cobra"
	"github.com/adrg/xdg"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start a Dorothy web server",
	Run: func(cmd *cobra.Command, args []string) {
		genconf, err := cmd.Flags().GetBool("genconf")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
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
		if err := cli.Serve(configpath, port, genconf); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolP("genconf", "g", false, "generate a default configuraiton file")
	serveCmd.Flags().IntP("port", "p", 4248, "port on which to listen")
	serveCmd.Flags().StringP(
		"config",
		"c",
		filepath.Join(xdg.ConfigHome, "dorothy", "config.toml"),
		"path to configuration file",
	)
}
