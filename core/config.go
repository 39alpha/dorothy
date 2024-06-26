package core

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	ma "github.com/multiformats/go-multiaddr"
)

type Config struct {
	User         *UserConfig     `toml:"user,omitempty"`
	Editor       string          `toml:"editor,omitempty"`
	RemoteString string          `toml:"remote,omitempty"`
	Ipfs         *IpfsConfig     `toml:"ipfs,omitempty"`
	Database     *DatabaseConfig `toml:"database,omitempty"`
	Remote       *Remote         `toml:"-"`
}

type Remote struct {
	Scheme       string
	Host         string
	Organization string
	Dataset      string
}

func NewRemote(remote string) (*Remote, error) {
	if remote == "" {
		return nil, nil
	}

	if !strings.HasPrefix(remote, "https://") && !strings.HasPrefix(remote, "http://") {
		return nil, fmt.Errorf("remote %q does not have http(s) scheme", remote)
	}

	u, err := url.Parse(remote)
	if err != nil {
		return nil, err
	}

	r := &Remote{}
	r.Scheme = u.Scheme
	r.Host = u.Host
	r.Organization, r.Dataset = filepath.Split(u.Path)
	r.Organization = path.Clean(r.Organization)
	_, r.Organization = filepath.Split(r.Organization)

	return r, nil
}

func (r Remote) BaseUrl() *url.URL {
	return &url.URL{
		Scheme: r.Scheme,
		Host:   r.Host,
	}
}

func (r Remote) Url() *url.URL {
	return r.BaseUrl().JoinPath(r.Organization, r.Dataset)
}

func (r Remote) UrlString() string {
	return r.Url().String()
}

func (r Remote) String() string {
	return r.UrlString()
}

type UserConfig struct {
	Name  string `toml:"name,omitempty"`
	Email string `toml:"email,omitempty"`
}

type IpfsConfig struct {
	Global bool   `toml:"global"`
	Host   string `toml:"host,omitempty"`
	Port   int    `toml:"port,omitempty"`
}

func (c IpfsConfig) Url() string {
	host := c.Host
	if host == "" {
		host = "http://127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 5001
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func (c IpfsConfig) Multiaddr() (ma.Multiaddr, error) {
	return ma.NewMultiaddr(c.Url())
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

func (u *UserConfig) String() string {
	s := u.Name
	if s != "" {
		s += " "
	}
	if u.Email != "" {
		s += "<" + u.Email + ">"
	}
	return s
}

func (config *Config) ReadFile(filename string) error {
	_, err := toml.DecodeFile(filename, config)
	if err != nil {
		return err
	}
	config.Remote, err = NewRemote(config.RemoteString)
	return err
}

func ReadConfigFile(filename string) (*Config, error) {
	var config Config
	if err := config.ReadFile(filename); err != nil {
		return nil, err
	}
	return &config, nil
}

func ReadConfigAsMap(filename string) (map[string]any, error) {
	var m map[string]any
	_, err := toml.DecodeFile(filename, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (config *Config) Read(r io.Reader) error {
	decoder := toml.NewDecoder(r)
	_, err := decoder.Decode(config)
	if err != nil {
		return err
	}
	config.Remote, err = NewRemote(config.RemoteString)
	return err
}

func ReadConfig(r io.Reader) (*Config, error) {
	var config Config
	if err := config.Read(r); err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *Config) WriteFile(filename string) error {
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer handle.Close()

	return config.Encode(handle)
}

func (config *Config) Encode(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	return encoder.Encode(config)
}
