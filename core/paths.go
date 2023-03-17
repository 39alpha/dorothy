package core

import "path/filepath"

const ROOT_PATH string = ".dorthy"

var configpath, manifestpath string

func init() {
	configpath = filepath.Join(ROOT_PATH, "config.json")
	manifestpath = filepath.Join(ROOT_PATH, "manifest.json")
}
