package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/39alpha/dorothy/core"
)

func Fetch() error {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration: %v", err)
	} else if config.RemoteString == "" {
		return fmt.Errorf("no remote set")
	} else if config.Remote == nil {
		return fmt.Errorf("ill-formed remote")
	}

	resp, err := http.Get(config.Remote.Url())
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var manifest core.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return err
	}

	return core.WriteManifestFile(MANIFEST_PATH, &manifest)
}
