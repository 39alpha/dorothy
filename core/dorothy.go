package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/kubo/core/coreiface/options"
)

type Dorothy struct {
	Config *Config
	Ipfs   *Ipfs

	Manifest *Manifest
}

func NewDorothy() (*Dorothy, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewDorothyFromConfig(config)
}

func NewDorothyFromConfigFile(filename string, noinherit bool) (*Dorothy, error) {
	config, err := LoadConfigFromFile(filename, noinherit)
	if err != nil {
		return nil, err
	}
	return NewDorothyFromConfig(config)
}

func NewDorothyFromConfig(config *Config) (*Dorothy, error) {
	ipfs, err := NewIpfs(config.Ipfs)
	if err != nil {
		return nil, err
	}

	manifest, _ := ReadManifestFile(filepath.Join(".dorothy", "manifest.json"))

	return &Dorothy{
		Config:   config,
		Ipfs:     ipfs,
		Manifest: manifest,
	}, nil
}

func (d *Dorothy) WriteConfig() error {
	return d.WriteConfigTo(filepath.Join(".dorothy", "config.toml"))
}

func (d *Dorothy) WriteConfigTo(filepath string) error {
	return d.Config.WriteFile(filepath)
}

func (d *Dorothy) WriteManifest() error {
	return d.WriteManifestTo(filepath.Join(".dorothy", "manifest.json"))
}

func (d *Dorothy) WriteManifestTo(filepath string) error {
	return d.Manifest.WriteFile(filepath)
}

func (d *Dorothy) InitializeDirectory(ctx context.Context, cwd string) error {
	dorothy_dir := filepath.Join(cwd, ".dorothy")

	if d.Manifest == nil {
		var err error
		d.Manifest, err = d.Ipfs.CreateEmptyManifest(ctx)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(dorothy_dir, 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorothy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorothy directory")
		}
	}

	config_path := filepath.Join(dorothy_dir, "config.toml")
	if err := d.WriteConfigTo(config_path); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	manifest_path := filepath.Join(dorothy_dir, "manifest.json")
	if err := d.WriteManifestTo(manifest_path); err != nil {
		return fmt.Errorf("failed to write manifest")
	}

	return nil
}

func (d *Dorothy) SetUserName(name string) error {
	if d.Config.User == nil {
		d.Config.User = &UserConfig{
			Name: name,
		}
	} else {
		d.Config.User.Name = name
	}

	return d.WriteConfig()
}

func (d *Dorothy) SetUserEmail(email string) error {
	if d.Config.User == nil {
		d.Config.User = &UserConfig{
			Email: email,
		}
	} else {
		d.Config.User.Email = email
	}

	return d.WriteConfig()
}

func (d *Dorothy) SetRemote(remote string) (err error) {
	d.Config.Remote, err = NewRemote(remote)
	if err != nil {
		return err
	}

	d.Config.RemoteString = d.Config.Remote.String()
	return d.WriteConfig()
}

func (d *Dorothy) SetEditor(editor string) error {
	d.Config.Editor = editor
	return d.WriteConfig()
}

func (d *Dorothy) Fetch() ([]Conflict, error) {
	if d.Config.RemoteString == "" {
		return nil, fmt.Errorf("no remote set")
	} else if d.Config.Remote == nil {
		return nil, fmt.Errorf("ill-formed remote")
	}

	req, err := http.NewRequest("GET", d.Config.Remote.Url(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}

	merged, conflicts, err := d.Manifest.Merge(&manifest)
	if err != nil || len(conflicts) != 0 {
		return conflicts, err
	}

	d.Manifest = merged
	return nil, d.WriteManifest()
}

func Clone(ctx context.Context, remote, dest string) (*Dorothy, error) {
	if dest == "" {
		r, err := NewRemote(remote)
		if err != nil {
			return nil, err
		}

		dest = r.Dataset
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return nil, fmt.Errorf("failed to create the repository directory %q", dest)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(dest); err != nil {
		return nil, err
	}
	defer os.Chdir(pwd)

	d, err := NewDorothy()
	if err != nil {
		return nil, err
	}

	if err := d.InitializeDirectory(ctx, "."); err != nil {
		return d, err
	}

	if err := d.SetRemote(remote); err != nil {
		return d, err
	}

	conflicts, err := d.Fetch()
	if len(conflicts) != 0 {
		return d, fmt.Errorf("clone encountered an unexpected dataset state")
	} else if err != nil {
		return d, err
	}

	return d, nil
}

func (d *Dorothy) Checkout(ctx context.Context, hash, dest string) error {
	if d.Manifest == nil {
		return fmt.Errorf("no manifest found")
	}

	for _, version := range d.Manifest.Versions {
		if version.Hash == hash {
			return d.Ipfs.Get(ctx, hash, dest)
		}
	}

	var matches []*Version
	for _, version := range d.Manifest.Versions {
		if strings.HasPrefix(version.Hash, hash) {
			matches = append(matches, version)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("hash %q not found in manifest", hash)
	} else if len(matches) == 1 {
		return d.Ipfs.Get(ctx, matches[0].Hash, dest)
	} else {
		return fmt.Errorf("hash matches multiple commits; aborting")
	}
}

func (d *Dorothy) Push() error {
	if d.Config.RemoteString == "" {
		return fmt.Errorf("no remote set")
	} else if d.Config.Remote == nil {
		return fmt.Errorf("ill-formed remote")
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.Encode(d.Manifest)

	handle, err := os.Open(filepath.Join(".dorothy", "manifest.json"))
	defer handle.Close()
	if err != nil {
		return fmt.Errorf("failed to read Manifest")
	}

	resp, err := http.Post(d.Config.Remote.Url(), "application/json", &buf)
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

	if err := json.Unmarshal(content, d.Manifest); err != nil {
		return fmt.Errorf("invalid Manifest received from the server: %v", err)
	}

	return d.WriteManifest()
}

func (d *Dorothy) Commit(ctx context.Context, path, message string, nopin bool, parents []string) (err error) {
	if d.Config.User == nil || d.Config.User.Name == "" || d.Config.User.Email == "" {
		return fmt.Errorf("user not configured; see `dorothy config user`")
	}

	if message == "" {
		return fmt.Errorf("empty message; aborting")
	}

	var hash string
	var pathtype PathType

	if stat, err := os.Stat(path); err != nil {
		return fmt.Errorf("cannot access dataset %q: %v", path, err)
	} else if stat.IsDir() {
		pathtype = PathTypeDirectory
		hash, err = d.Ipfs.Add(ctx, path, options.Unixfs.Pin(!nopin), options.Unixfs.Progress(true))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	} else {
		pathtype = PathTypeFile
		hash, err = d.Ipfs.Add(ctx, path, options.Unixfs.Pin(!nopin))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	}

	version := &Version{
		Author:   d.Config.User.String(),
		Date:     time.Now(),
		Message:  message,
		Hash:     hash,
		PathType: pathtype,
		Parents:  parents,
	}

	var manifest *Manifest
	var conflicts []Conflict
	manifest, conflicts, err = d.Manifest.Merge(
		&Manifest{
			Versions: []*Version{version},
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

	d.Manifest = manifest
	return d.WriteManifest()
}

func (d *Dorothy) UnknownCommits(commits []string) []string {
	return d.Manifest.UnknownCommits(commits)
}

func (d *Dorothy) ReadFromEditor(filename string) (string, error) {
	editor := d.Config.Editor
	if editor == "" {
		var ok bool
		if editor, ok = os.LookupEnv("EDITOR"); !ok {
			return "", fmt.Errorf("cannot find a suitable text editor")
		}
	}

	path := filepath.Join(".dorothy", filename)
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

	body, err := os.ReadFile(path)
	if len(body) == 0 {
		return "", fmt.Errorf("no content found in %q", path)
	}
	body = bytes.TrimRight(body, "\n\r")

	return string(body), nil
}

func (d *Dorothy) Recieve(ctx context.Context, old, new *Manifest) (*Manifest, []Conflict, error) {
	merged, conflicts, err := old.Merge(new)
	if err != nil || len(conflicts) != 0 {
		return nil, conflicts, err
	}

	merged, err = d.Ipfs.SaveManifest(ctx, merged)
	if err != nil {
		return nil, nil, err
	}

	return merged, nil, nil
}
