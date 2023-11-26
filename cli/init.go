package cli

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core/model"
)

func Init() error {
	if err := os.Mkdir(".dorothy", 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorothy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorothy directory")
		}
	}

	if err := (&Config{}).WriteConfigFile(configpath); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := model.WriteManifestFile(manifestpath, &model.Manifest{}); err != nil {
		return fmt.Errorf("failed to open manifest")
	}

	return nil
}
