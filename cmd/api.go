package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/39alpha/dorothy/cli"
	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "start the api server",
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
		if err := cli.ServeApi(configpath, port, genconf); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serveCmd.AddCommand(apiCmd)
	apiCmd.Flags().BoolP("genconf", "g", false, "generate a default configuraiton file")
	apiCmd.Flags().IntP("port", "p", 4248, "port on which to listen")
	apiCmd.Flags().StringP(
		"config",
		"c",
		filepath.Join(xdg.ConfigHome, "dorothy", "config.toml"),
		"path to configuration file",
	)
}
