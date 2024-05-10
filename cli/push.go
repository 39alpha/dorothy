package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
)

func Push() error {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	} else if config.RemoteString == "" {
		return fmt.Errorf("no remote set")
	} else if config.Remote == nil {
		return fmt.Errorf("ill-formed remote")
	}

	handle, err := os.Open(MANIFEST_PATH)
	defer handle.Close()
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	resp, err := http.Post(config.Remote.Url(), "application/json", handle)
	if err != nil {
		return err
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot process server response")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("push failed: %s", content)
	}

	var manifest model.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return fmt.Errorf("invalid manifest received from the server: %v", err)
	}

	return model.WriteManifestFile(MANIFEST_PATH, &manifest)
}
