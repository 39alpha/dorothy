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
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/ipfs/kubo/core/coreiface/options"
)

type Dorothy struct {
	context.Context
	Directory     string
	LoadedConfigs []string
	Config        Config
	Ipfs          Ipfs
	Manifest      *Manifest
}

func NewDorothy() (*Dorothy, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dorothy := &Dorothy{
		Context:       context.Background(),
		Directory:     filepath.Join(cwd, ".dorothy"),
		LoadedConfigs: []string{},
	}

	if err := dorothy.LoadDefaultConfig(); err != nil {
		return nil, err
	}

	return dorothy, dorothy.ReloadIpfs()
}

func (d *Dorothy) Setup(options ...IpfsNodeOption) error {
	if !d.IsInitialized() {
		return fmt.Errorf("not a dorothy repository")
	}

	if err := d.ConnectIpfs(options...); err != nil {
		return err
	}

	return d.LoadManifest()
}

func (d *Dorothy) GlobalConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "dorothy", "config.toml")
}

func (d *Dorothy) LocalConfigPath() string {
	return filepath.Join(d.Directory, "config.toml")
}

func (d *Dorothy) ManifestPath() string {
	return filepath.Join(d.Directory, "manifest")
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

func (d *Dorothy) ResetConfig() error {
	d.LoadedConfigs = []string{}
	return d.ReloadConfig()
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
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	d.LoadedConfigs = append(d.LoadedConfigs, filename)

	return d.ReloadIpfs()
}

func (d *Dorothy) ReloadConfig() error {
	d.Config = Config{}
	for _, filename := range d.LoadedConfigs {
		if err := d.LoadConfigFile(filename); err != nil {
			return err
		}
	}
	return d.ReloadIpfs()
}

func (d *Dorothy) ReloadIpfs() error {
	connected := d.Ipfs.IsConnected()
	d.Ipfs = NewIpfs(d.Config.Ipfs)
	if connected {
		return d.ConnectIpfs()
	}
	return nil
}

func (d *Dorothy) LoadManifest() error {
	if !d.Ipfs.IsConnected() {
		return fmt.Errorf("not connected to IPFS")
	}

	hash, err := os.ReadFile(d.ManifestPath())
	if err != nil {
		return err
	}

	d.Manifest, err = d.Ipfs.GetManifest(d, string(hash))
	return err
}

func (d *Dorothy) WriteManifestFile() error {
	if d.Manifest == nil {
		return fmt.Errorf("no manifest loaded")
	}

	return os.WriteFile(d.ManifestPath(), []byte(d.Manifest.Hash), 0755)
}

func (d *Dorothy) InitializeIpfs() error {
	return d.Ipfs.Initialize(d.Directory)
}

func (d *Dorothy) ConnectIpfs(options ...IpfsNodeOption) error {
	return d.Ipfs.Connect(d, d.Directory, options...)
}

func (d *Dorothy) WriteConfig() error {
	return d.Config.WriteFile(d.LocalConfigPath())
}

func (d *Dorothy) InitializeAndConnectIpfs(options ...IpfsNodeOption) error {
	if err := d.InitializeIpfs(); err != nil {
		return fmt.Errorf("failed to initialize IPFS: %v", err)
	}

	if err := d.ConnectIpfs(options...); err != nil {
		return fmt.Errorf("failed to connect to IPFS")
	}

	return nil
}

func (d *Dorothy) Initialize(global bool, options ...IpfsNodeOption) error {
	if d.IsInitialized() {
		return d.InitializeAndConnectIpfs(options...)
	}

	if err := os.MkdirAll(d.Directory, 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("dorothy already initialized")
		} else {
			return fmt.Errorf("failed to create the .dorothy directory")
		}
	}

	config := Config{}
	if global {
		config.Ipfs = &IpfsConfig{
			Global: true,
		}
		if d.Config.Ipfs == nil {
			d.Config.Ipfs = config.Ipfs
		} else {
			d.Config.Ipfs.Global = true
		}
		d.Ipfs = NewIpfs(d.Config.Ipfs)
	}

	if err := (&config).WriteFile(d.LocalConfigPath()); err != nil {
		return fmt.Errorf("failed to write configuration")
	}

	if err := d.InitializeAndConnectIpfs(options...); err != nil {
		return err
	}

	var err error
	if d.Manifest == nil {
		d.Manifest, err = d.Ipfs.CreateEmptyManifest(d)
	} else {
		d.Manifest, err = d.Ipfs.SaveManifest(d, d.Manifest)
	}
	if err != nil {
		return err
	}

	if err := d.WriteManifestFile(); err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}

	return nil
}

func promote(x string) any {
	if value, err := strconv.ParseBool(x); err == nil {
		return value
	}

	if value, err := strconv.ParseInt(x, 10, 64); err == nil {
		return value
	}

	if value, err := strconv.ParseFloat(x, 64); err == nil {
		return value
	}

	return x
}

func (d *Dorothy) SetConfig(props []string, value string, global bool) (string, error) {
	var err error
	var m map[string]any

	var configpath string
	if global {
		configpath = d.GlobalConfigPath()
	} else {
		configpath = d.LoadedConfigs[len(d.LoadedConfigs)-1]
	}
	m, err = ReadConfigAsMap(configpath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		m = map[string]any{}
	}

	n := len(props)
	s := m
	for i, prop := range props {
		if i == n-1 {
			s[prop] = promote(value)
			break
		}

		if v, ok := s[prop]; ok {
			s, ok = v.(map[string]any)
			if !ok {
				return "", fmt.Errorf("invalid property")
			}
		} else {
			v = map[string]any{}
			s[prop] = v
			s = v.(map[string]any)
		}
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(m); err != nil {
		return "", err
	}

	config := Config{}
	decoder := toml.NewDecoder(&buf)
	if _, err := decoder.Decode(&config); err != nil {
		return "", err
	}

	if err = config.WriteFile(configpath); err != nil {
		if errors.Is(err, os.ErrNotExist) && global {
			if err := os.MkdirAll(filepath.Dir(configpath), 0755); err != nil {
				return "", err
			}
			handle, err := os.Create(configpath)
			if err != nil {
				return "", err
			}
			handle.Close()
			err = config.WriteFile(configpath)
		} else {
			return "", err
		}
	}

	return configpath, d.ReloadConfig()
}

func (d *Dorothy) GetConfig(props []string) (any, error) {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(d.Config); err != nil {
		return nil, err
	}

	var m map[string]any
	decoder := toml.NewDecoder(&buf)
	if _, err := decoder.Decode(&m); err != nil {
		return nil, err
	}

	n := len(props)
	for i, prop := range props {
		v, ok := m[prop]
		if !ok {
			break
		}

		if i == n-1 {
			return v, nil
		}

		m, ok = v.(map[string]any)
		if !ok {
			break
		}
	}

	return nil, fmt.Errorf("configuration property not found")
}

func (d *Dorothy) DelConfig(props []string, global bool) (string, error) {
	var err error
	var m map[string]any

	var configpath string
	if global {
		configpath = d.GlobalConfigPath()
	} else {
		configpath = d.LoadedConfigs[len(d.LoadedConfigs)-1]
	}
	m, err = ReadConfigAsMap(configpath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		m = map[string]any{}
	}

	n := len(props)
	s := m
	for i, prop := range props {
		if i == n-1 {
			if _, ok := s[prop]; ok {
				delete(s, prop)
				break
			} else {
				return "", fmt.Errorf("invalid property")
			}
		}

		if v, ok := s[prop]; ok {
			s, ok = v.(map[string]any)
			if !ok {
				return "", fmt.Errorf("invalid property")
			}
		} else {
			v = map[string]any{}
			s[prop] = v
			s = v.(map[string]any)
		}
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(m); err != nil {
		return "", err
	}

	config := Config{}
	decoder := toml.NewDecoder(&buf)
	if _, err := decoder.Decode(&config); err != nil {
		return "", err
	}

	if err = config.WriteFile(configpath); err != nil {
		if errors.Is(err, os.ErrNotExist) && global {
			if err := os.MkdirAll(filepath.Dir(configpath), 0755); err != nil {
				return "", err
			}
			handle, err := os.Create(configpath)
			if err != nil {
				return "", err
			}
			handle.Close()
			err = config.WriteFile(configpath)
		} else {
			return "", err
		}
	}

	return configpath, d.ReloadConfig()
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

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if err := d.Ipfs.ConnectToPeerById(d, payload.PeerIdentity); err != nil {
		return nil, err
	}

	manifest, err := d.Ipfs.GetManifest(d, payload.Hash)
	if err != nil {
		return nil, err
	}

	merged, conflicts, err := d.Ipfs.MergeAndCommit(d, d.Manifest, manifest)
	if err != nil || len(conflicts) != 0 {
		return conflicts, err
	}

	d.Manifest = merged
	return nil, d.WriteManifestFile()
}

func Clone(remote, dest string, global bool) (*Dorothy, error) {
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

	err = os.Chdir(dest)
	if err != nil {
		return nil, err
	}
	defer os.Chdir(cwd)

	d, err := NewDorothy()
	if err != nil {
		return nil, err
	}

	if d.IsInitialized() {
		return nil, fmt.Errorf("directory already contains an initialized dataset")
	}

	if err := d.Initialize(global); err != nil {
		return d, err
	}

	if _, err := d.SetConfig([]string{"remote"}, remote, false); err != nil {
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
	if !d.Ipfs.IsConnected() {
		return fmt.Errorf("not connected to IPFS")
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

func (d *Dorothy) Push() ([]Conflict, error) {
	if !d.Ipfs.IsConnected() {
		return nil, fmt.Errorf("not connected to IPFS")
	}

	if d.Config.RemoteString == "" {
		return nil, fmt.Errorf("no remote set")
	} else if d.Config.Remote == nil {
		return nil, fmt.Errorf("ill-formed remote")
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.Encode(Payload{
		Hash:         d.Manifest.Hash,
		PeerIdentity: d.Ipfs.Identity,
	})

	resp, err := http.Post(d.Config.Remote.Url(), "application/json", &buf)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot process server response")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("push failed: %s", content)
	}

	var payload Payload
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("invalid response received from the server: %v", err)
	}

	if err := d.Ipfs.ConnectToPeerById(d, payload.PeerIdentity); err != nil {
		return nil, fmt.Errorf("failed to connect to remote peer after push: %v", err)
	}

	remote, err := d.Ipfs.GetManifest(d, payload.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve manifest after push: %v", err)
	}

	merged, conflicts, err := d.Ipfs.MergeAndCommit(d, d.Manifest, remote)
	if err != nil || len(conflicts) != 0 {
		return conflicts, err
	}

	d.Manifest = merged
	return nil, d.WriteManifestFile()
}

func (d *Dorothy) Commit(path, message string, nopin bool, parents []string) ([]Conflict, error) {
	if !d.Ipfs.IsConnected() {
		return nil, fmt.Errorf("not connected to IPFS")
	}

	if d.Config.User == nil || d.Config.User.Name == "" || d.Config.User.Email == "" {
		return nil, fmt.Errorf("user not configured; see `dorothy config user`")
	}

	if message == "" {
		return nil, fmt.Errorf("empty message; aborting")
	}

	var hash string
	var pathtype PathType

	if stat, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("cannot access dataset %q: %v", path, err)
	} else if stat.IsDir() {
		pathtype = PathTypeDirectory
	} else {
		pathtype = PathTypeFile
	}

	hash, err := d.Ipfs.Add(d, path, options.Unixfs.Pin(!nopin), options.Unixfs.Progress(true))
	if err != nil {
		return nil, fmt.Errorf("failed to add dataset %q: %v", path, err)
	}

	merged, conflicts, err := d.Ipfs.MergeAndCommit(d, d.Manifest, &Manifest{
		Versions: []*Version{
			{
				Author:   d.Config.User.String(),
				Date:     time.Now(),
				Message:  message,
				Hash:     hash,
				PathType: pathtype,
				Parents:  parents,
			},
		},
	})

	if err != nil || len(conflicts) != 0 {
		return conflicts, err
	}

	d.Manifest = merged
	return nil, d.WriteManifestFile()
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

func (d *Dorothy) Recieve(old *Manifest, hash string) (*Manifest, []Conflict, error) {
	if !d.Ipfs.IsConnected() {
		return nil, nil, fmt.Errorf("not connected to IPFS")
	}

	new, err := d.Ipfs.GetManifest(d, hash)
	if err != nil {
		return nil, nil, err
	}

	return d.Ipfs.MergeAndCommit(d, old, new)
}
