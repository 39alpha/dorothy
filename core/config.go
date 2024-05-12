package core

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
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
	Host         string
	Organization string
	Dataset      string
}

func NewRemote(remote string) (*Remote, error) {
	if remote == "" {
		return nil, nil
	}

	r := &Remote{}

	u, err := url.Parse(remote)
	if err != nil {
		return r, err
	}
	r.Host = u.Host
	r.Organization, r.Dataset = filepath.Split(u.Path)
	r.Organization = path.Clean(r.Organization)
	_, r.Organization = filepath.Split(r.Organization)

	return r, nil
}

func (r Remote) String() string {
	return "dorothy://" + r.Host + "/" + r.Organization + "/" + r.Dataset
}

func (r Remote) Url() string {
	return "http://" + r.Host + "/" + r.Organization + "/" + r.Dataset
}

type UserConfig struct {
	Name  string `toml:"name,omitempty"`
	Email string `toml:"email,omitempty"`
}

type IpfsConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (c IpfsConfig) Url() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
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

func (config *Config) Read(r io.Reader) error {
	decoder := toml.NewDecoder(r)
	_, err := decoder.Decode(config)
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
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return err
	}

	return config.Encode(handle)
}

func (config *Config) Encode(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	return encoder.Encode(config)
}

func LoadConfig() (*Config, error) {
	paths := []string{
		filepath.Join(xdg.ConfigHome, "dorothy", "config.toml"),
		filepath.Join(".dorothy", "config.toml"),
	}

	var config Config
	for _, configpath := range paths {
		if err := config.ReadFile(configpath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return &config, nil
}

func LoadConfigFromFile(filename string, noinherit bool) (*Config, error) {
	if noinherit {
		return ReadConfigFile(filename)
	} else {
		if config, err := LoadConfig(); err != nil {
			return nil, err
		} else if err := config.ReadFile(filename); err != nil {
			return nil, err
		} else {
			return config, nil
		}
	}
}
