package cli

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
)

func Init() error {
	if err := os.Mkdir(".dorothy", 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorothy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorothy directory")
		}
	}

	if err := (&core.Config{}).WriteConfigFile(CONFIG_PATH); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := core.WriteManifestFile(MANIFEST_PATH, &core.Manifest{}); err != nil {
		return fmt.Errorf("failed to open manifest")
	}

	return nil
}
