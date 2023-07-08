package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/39alpha/dorothy/core"
)

func Fetch() error {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration: %v", err)
	} else if config.Remote == "" {
		return fmt.Errorf("no remote set")
	}

	r, err := ParseRemote(config.Remote)
	if err != nil {
		return err
	}

	resp, err := http.Get(r.Url())
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

	return core.WriteManifestFile(manifestpath, manifest)
}
