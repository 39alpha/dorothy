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

	if err := (&core.Config{}).WriteFile(CONFIG_PATH); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := (&core.Manifest{}).WriteFile(MANIFEST_PATH); err != nil {
		return fmt.Errorf("failed to open manifest")
	}

	return nil
}
