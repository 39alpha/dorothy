package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	ipfs "github.com/ipfs/go-ipfs-api"
)

func CommitData(path, message string, nopin bool, parents []string, pick bool) (err error) {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	} else if config.User == nil || config.User.Name == "" || config.User.Email == "" {
		return fmt.Errorf("user not configured; see `dorothy config user`")
	}

	if message == "" {
		message, err = FromEditor(config, "commit-msg")
		if err != nil {
			return fmt.Errorf("%v; aborting commit", err)
		}
	}

	manifest, err := model.ReadManifestFile(MANIFEST_PATH)
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	parents, ok, err := checkParentage(manifest, parents, pick)
	if err != nil {
		return fmt.Errorf("%v; aborting commit", err)
	} else if !ok {
		return nil
	}

	var hash string
	var pathtype model.PathType

	client := ipfs.NewLocalShell()
	if stat, err := os.Stat(path); err != nil {
		return fmt.Errorf("cannot access dataset %q: %v", path, err)
	} else if stat.IsDir() {
		pathtype = model.PathTypeDirectory
		hash, err = client.AddDir(path, ipfs.Pin(!nopin), ipfs.Progress(true))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	} else {
		pathtype = model.PathTypeFile
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

	version := &model.Version{
		Author:   config.User.String(),
		Date:     time.Now(),
		Message:  message,
		Hash:     hash,
		PathType: pathtype,
		Parents:  parents,
	}

	var conflicts []model.Conflict
	manifest, conflicts, err = manifest.Merge(
		&model.Manifest{
			Versions: []*model.Version{version},
		},
	)
	if len(conflicts) != 0 {
		s := strings.Builder{}
		s.WriteString("Version conflicts with one ore more previous versions:\n\n")
		for _, conflict := range conflicts {
			s.WriteString(conflict.String())
		}
		return fmt.Errorf(s.String())
	} else if err != nil {
		return err
	}

	return model.WriteManifestFile(MANIFEST_PATH, manifest)
}

func FromEditor(config *core.Config, filename string) (string, error) {
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

func checkParentage(manifest *model.Manifest, parents []string, pick bool) ([]string, bool, error) {
	if len(manifest.Versions) != 0 && (pick || len(parents) == 0) {
		var picked []string
		for {
			var err error
			picked, err = chooseVersions(
				"Which versions are parents of this version?",
				manifest,
				false,
			)
			if err != nil {
				return nil, false, err
			} else if len(picked) == 0 {
				fmt.Print("No parents selected. Do you want to continue (y/N) ")
				var res string
				fmt.Scanln(&res)
				if ok, err := regexp.MatchString("(?i)^y(es)?$", res); err == nil && ok {
					break
				}
			} else {
				break
			}
		}
		parents = append(parents, picked...)
	}

	var unknown []string
	for _, parent := range parents {
		seen := false
		for _, version := range manifest.Versions {
			if version.Hash == parent {
				seen = true
				break
			}
		}
		if !seen {
			unknown = append(unknown, parent)
		}
	}

	if len(unknown) != 0 {
		fmt.Println("The following parents are not in the manifest")
		for _, parent := range unknown {
			fmt.Printf("  %s\n", parent)
		}
		for {
			fmt.Print("Do you want to continue (y/N) ")
			var res string
			fmt.Scanln(&res)
			if ok, err := regexp.MatchString("(?i)^y(es)?$", res); err == nil && ok {
				return parents, true, nil
			} else if ok, err := regexp.MatchString("(?i)^n(o)?$", res); err == nil && ok {
				return nil, false, nil
			}
		}
	}

	sort.Strings(parents)

	return parents, true, nil
}
