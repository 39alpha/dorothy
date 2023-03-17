package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func Push() error {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	} else if config.Remote == "" {
		return fmt.Errorf("no remote set")
	}

	r, err := ParseRemote(config.Remote)
	if err != nil {
		return err
	}

	handle, err := os.Open(manifestpath)
	defer handle.Close()
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	resp, err := http.Post(r.Url(), "application/json", handle)
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

	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return fmt.Errorf("invalid manifest received from the server: %v", err)
	}

	return WriteManifestFile(manifestpath, manifest)
}
