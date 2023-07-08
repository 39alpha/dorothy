package cli

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
)

func ServeApi(configpath string, port int, genconf bool) error {
	if genconf {
		if err := core.GenerateConfig(os.Stdout); err != nil {
			return fmt.Errorf("failed to generate a config\n  %v", err)
		}
	} else {
		config, err := core.ReadConfig(configpath)
		if err != nil {
			return fmt.Errorf("Error: invalid configuration file %q\n  %v\n", configpath, err)
		}

		app, err := core.NewDorthy(config)
		if err != nil {
			return fmt.Errorf("Error: failed to start Dorthy\n  %v\n", err)
		}

		app.Listen(fmt.Sprintf(":%d", port))
	}

	return nil
}
