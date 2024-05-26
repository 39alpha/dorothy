package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/ipfs/kubo/core/coreiface/options"
)

type Dorothy struct {
	context.Context
	Directory     string
	LoadedConfigs []string
	Config        Config
	Ipfs          *Ipfs
	Manifest      *Manifest
}

func NewDorothy() (*Dorothy, bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false, err
	}

	dorothy := &Dorothy{
		Context:       context.Background(),
		Directory:     filepath.Join(cwd, ".dorothy"),
		LoadedConfigs: []string{},
		Config:        &Config{},
	}

	if err := dorothy.LoadDefaultConfig(); err != nil {
		return nil, false, err
	}

	dorothy.Ipfs = NewIpfs(dorothy.Config.Ipfs)

	if !dorothy.IsInitialized() {
		return dorothy, false, nil
	}

	if err := dorothy.ConnectIpfs(); err != nil {
		return nil, true, err
	}

	if err := dorothy.LoadManifest(); err != nil {
		return nil, true, err
	}

	return dorothy, true, nil
}

func (d *Dorothy) GlobalConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "dorothy", "config.toml")
}

func (d *Dorothy) LocalConfigPath() string {
	return filepath.Join(d.Directory, "config.toml")
}

func (d *Dorothy) ManifestPath() string {
	return filepath.Join(d.Directory, "manifest.json")
}

func (d *Dorothy) IsInitialized() bool {
	expected_paths := []struct {
		path   string
		is_dir bool
	}{
		{path: d.Directory, is_dir: true},
		{path: d.LocalConfigPath(), is_dir: false},
		{path: d.ManifestPath(), is_dir: false},
	}
	for _, expected := range expected_paths {
		stat, err := os.Stat(expected.path)
		if err != nil {
			return false
		}

		if stat.IsDir() != expected.is_dir {
			return false
		}
	}

	return true
}

func (d *Dorothy) ResetConfig() {
	d.Config = Config{}
	d.LoadedConfigs = []string{}
}

func (d *Dorothy) LoadGlobalConfig() error {
	return d.LoadConfigFile(d.GlobalConfigPath())
}

func (d *Dorothy) LoadLocalConfig() error {
	return d.LoadConfigFile(d.LocalConfigPath())
}

func (d *Dorothy) LoadDefaultConfig() error {
	paths := []string{
		d.GlobalConfigPath(),
		d.LocalConfigPath(),
	}

	for _, configpath := range paths {
		if err := d.LoadConfigFile(configpath); err != nil {
			return err
		}
	}

	return nil
}

func (d *Dorothy) LoadConfigFile(filename string) error {
	if err := d.Config.ReadFile(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			return err
		}
	}

	d.LoadedConfigs = append(d.LoadedConfigs, filename)

	return nil
}

func (d *Dorothy) ReloadConfig() error {
	d.Config = Config{}
	for _, filename := range d.LoadedConfigs {
		if err := d.LoadConfigFile(filename); err != nil {
			return err
		}
	}
	return d.ReconnectIpfs()
}

func (d *Dorothy) ReconnectIpfs() error {
	if d.Config.Ipfs != nil && d.Ipfs != nil && d.Ipfs.CoreAPI != nil {
		d.Ipfs = NewIpfs(d.Config.Ipfs)
		return d.ConnectIpfs()
	}
	return nil
}

func (d *Dorothy) LoadManifest() error {
	var err error

	d.Manifest, err = ReadManifestFile(d.ManifestPath())

	return err
}

func (d *Dorothy) InitializeIpfs() error {
	return d.Ipfs.Initialize(d.Directory)
}

func (d *Dorothy) ConnectIpfs() error {
	return d.Ipfs.Connect(d, d.Directory)
}

func (d *Dorothy) WriteConfig() error {
	return d.Config.WriteFile(d.LocalConfigPath())
}

func (d *Dorothy) WriteManifest() error {
	return d.Manifest.WriteFile(d.ManifestPath())
}

func (d *Dorothy) InitializeAndConnectIpfs() error {
	if err := d.InitializeIpfs(); err != nil {
		return fmt.Errorf("failed to initialize IPFS: %v", err)
	}

	if err := d.ConnectIpfs(); err != nil {
		return fmt.Errorf("failed to connect to IPFS")
	}
	return nil

}

func (d *Dorothy) Initialize() error {
	if d.IsInitialized() {
		return d.InitializeAndConnectIpfs()
	}

	if err := os.MkdirAll(d.Directory, 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorothy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorothy directory")
		}
	}

	if err := (&Config{}).WriteFile(d.LocalConfigPath()); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := d.InitializeAndConnectIpfs(); err != nil {
		return err
	}

	if d.Manifest == nil {
		var err error
		d.Manifest, err = d.Ipfs.CreateEmptyManifest(d)
		if err != nil {
			return err
		}
	}

	if err := d.WriteManifest(); err != nil {
		return fmt.Errorf("failed to write manifest")
	}

	return nil
}

func (d *Dorothy) SetUserName(name string) error {
	config, err := ReadConfigFile(d.LocalConfigPath())
	if err != nil {
		return nil
	}

	if config.User == nil {
		config.User = &UserConfig{
			Name: name,
		}
	} else {
		config.User.Name = name
	}

	if err := config.WriteFile(d.LocalConfigPath()); err != nil {
		return err
	}

	return d.ReloadConfig()
}

func (d *Dorothy) SetUserEmail(email string) error {
	config, err := ReadConfigFile(d.LocalConfigPath())
	if err != nil {
		return nil
	}

	if config.User == nil {
		config.User = &UserConfig{
			Email: email,
		}
	} else {
		config.User.Email = email
	}

	if err := config.WriteFile(d.LocalConfigPath()); err != nil {
		return err
	}

	return d.ReloadConfig()
}

func (d *Dorothy) SetRemote(remote string) (err error) {
	config, err := ReadConfigFile(d.LocalConfigPath())
	if err != nil {
		return nil
	}

	config.Remote, err = NewRemote(remote)
	if err != nil {
		return err
	}

	config.RemoteString = config.Remote.String()

	if err := config.WriteFile(d.LocalConfigPath()); err != nil {
		return err
	}

	return d.ReloadConfig()
}

func (d *Dorothy) SetEditor(editor string) error {
	config, err := ReadConfigFile(d.LocalConfigPath())
	if err != nil {
		return nil
	}

	config.Editor = editor

	if err := config.WriteFile(d.LocalConfigPath()); err != nil {
		return err
	}

	return d.ReloadConfig()
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

func Clone(remote, dest string) (*Dorothy, error) {
	if dest == "" {
		r, err := NewRemote(remote)
		if err != nil {
			return nil, err
		}

		dest = r.Dataset
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dest = filepath.Join(cwd, dest)

	if err := os.MkdirAll(dest, 0755); err != nil {
		return nil, fmt.Errorf("failed to create the repository directory %q", dest)
	}

	d, initialized, err := NewDorothy()
	if err != nil {
		return nil, err
	} else if initialized {
		return nil, fmt.Errorf("directory already contains an initialized dataset")
	}
	d.Directory = dest

	if err := d.Initialize(); err != nil {
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

func (d *Dorothy) Checkout(hash, dest string) error {
	if err := d.ConnectIpfs(); err != nil {
		return err
	}

	if d.Manifest == nil {
		return fmt.Errorf("no manifest found")
	}

	for _, version := range d.Manifest.Versions {
		if version.Hash == hash {
			return d.Ipfs.Get(d, hash, dest)
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
		return d.Ipfs.Get(d, matches[0].Hash, dest)
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

func (d *Dorothy) Commit(path, message string, nopin bool, parents []string) (err error) {
	if d.Config.User == nil || d.Config.User.Name == "" || d.Config.User.Email == "" {
		return fmt.Errorf("user not configured; see `dorothy config user`")
	}

	if message == "" {
		return fmt.Errorf("empty message; aborting")
	}

	if err := d.ConnectIpfs(); err != nil {
		return err
	}

	var hash string
	var pathtype PathType

	if stat, err := os.Stat(path); err != nil {
		return fmt.Errorf("cannot access dataset %q: %v", path, err)
	} else if stat.IsDir() {
		pathtype = PathTypeDirectory
		hash, err = d.Ipfs.Add(d, path, options.Unixfs.Pin(!nopin), options.Unixfs.Progress(true))
		if err != nil {
			return fmt.Errorf("failed to add dataset %q: %v", path, err)
		}
	} else {
		pathtype = PathTypeFile
		hash, err = d.Ipfs.Add(d, path, options.Unixfs.Pin(!nopin))
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

func (d *Dorothy) Recieve(old, new *Manifest) (*Manifest, []Conflict, error) {
	if err := d.ConnectIpfs(); err != nil {
		return nil, nil, err
	}

	merged, conflicts, err := old.Merge(new)
	if err != nil || len(conflicts) != 0 {
		return nil, conflicts, err
	}

	merged, err = d.Ipfs.SaveManifest(d, merged)
	if err != nil {
		return nil, nil, err
	}

	return merged, nil, nil
}
