package cli

import (
	"fmt"
	"os"

	"github.com/39alpha/dorthy/core"
)

func Init() error {
	if err := os.Mkdir(".dorthy", 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorthy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorthy directory")
		}
	}

	if err := (&Config{}).WriteConfigFile(configpath); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := core.WriteManifestFile(manifestpath, core.Manifest{}); err != nil {
		return fmt.Errorf("failed to open manifest")
	}

	return nil
}
