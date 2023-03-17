package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	ipfs "github.com/ipfs/go-ipfs-api"
)

func CommitData(path, message string, nopin bool) (err error) {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	} else if config.User == nil || config.User.Name == "" || config.User.Email == "" {
		return fmt.Errorf("user not configured; see `dorthy config user`")
	}

	if message == "" {
		message, err = FromEditor(config, "commit-msg")
		if err != nil {
			return fmt.Errorf("%v; aborting commit", err)
		}
	}

	manifest, err := ReadManifestFile(manifestpath)
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	var hash string
	var pathtype PathType

	client := ipfs.NewLocalShell()
	if stat, err := os.Stat(path); err != nil {
		return fmt.Errorf("cannot access dataset %q: %v", path, err)
	} else if stat.IsDir() {
		pathtype = D_DIR
		hash, err = client.AddDir(path, ipfs.Pin(!nopin), ipfs.Progress(true))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	} else {
		pathtype = D_FILE
		handle, err := os.Open(path)
		defer handle.Close()
		if err != nil {
			return fmt.Errorf("failed to open dataset %q: %v", path, err)
		}

		hash, err = client.Add(handle, ipfs.Pin(!nopin))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	}

	for _, entry := range manifest {
		if entry.Hash == hash {
			return fmt.Errorf("version is already tracked; aborting")
		}
	}

	commit := Commit{
		Author:    config.User.String(),
		Date:      time.Now(),
		Message:   message,
		Hash:      hash,
		Type:      pathtype,
		Ancestors: []string{},
	}

	manifest = append(manifest, commit)
	return WriteManifestFile(manifestpath, manifest)
}

func FromEditor(config *Config, filename string) (string, error) {
	editor := config.Editor
	if editor == "" {
		var ok bool
		if editor, ok = os.LookupEnv("EDITOR"); !ok {
			return "", fmt.Errorf("cannot find a suitable text editor")
		}
	}

	path := filepath.Join(ROOT_PATH, filename)
	defer os.RemoveAll(path)

	handle, err := os.Create(path)
	handle.Close()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor crashed")
	}

	body, err := ioutil.ReadFile(path)
	if len(body) == 0 {
		return "", fmt.Errorf("no content found in %q", path)
	}
	body = bytes.TrimRight(body, "\n\r")

	return string(body), nil
}
