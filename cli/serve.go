package cli

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server"
)

func Serve(configpath string, port int) error {
	config, err := core.ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("Error: invalid configuration file %q\n  %v\n", configpath, err)
	}

	app, err := server.NewServer(config)
	if err != nil {
		return fmt.Errorf("Error: failed to start Dorothy\n  %v\n", err)
	}

	return app.Listen(fmt.Sprintf(":%d", port))
}
