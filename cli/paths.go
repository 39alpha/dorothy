package cli

import "path/filepath"

const ROOT_PATH string = ".dorothy"

var CONFIG_PATH, MANIFEST_PATH string

func init() {
	CONFIG_PATH = filepath.Join(ROOT_PATH, "config.toml")
	MANIFEST_PATH = filepath.Join(ROOT_PATH, "manifest.json")
}
