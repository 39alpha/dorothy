package core

import (
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
	Filename     string        `toml:"-"`
	User         *UserConfig   `toml:"user,omitempty"`
	Editor       string        `toml:"editor,omitempty"`
	RemoteString string        `toml:"remote,omitempty"`
	Ipfs         *IpfsConfig   `toml:"ipfs,omitempty"`
	Server       *ServerConfig `toml:"server,omitempty"`
	Remote       *Remote       `toml:"-"`
}

type ServerConfig struct {
	Ipfs     *IpfsConfig     `toml:"ipfs,omitempty"`
	Database *DatabaseConfig `toml:"database,omitempty"`
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

func ReadConfigFile(filename string) (*Config, error) {
	config := Config{
		Filename: filename,
	}

	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return nil, err
	}

	config.Remote, err = NewRemote(config.RemoteString)

	return &config, nil
}

func ReadConfig(r io.Reader) (*Config, error) {
	var config Config
	decoder := toml.NewDecoder(r)
	_, err := decoder.Decode(&config)
	return &config, err
}

func (config *Config) WriteConfigFile(filename string) error {
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return err
	}

	return config.WriteConfig(handle)
}

func (config *Config) WriteConfig(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	return encoder.Encode(config)
}

func GenerateConfig(w io.Writer) error {
	config := Config{
		Filename: "",
		Ipfs: &IpfsConfig{
			Host: "127.0.0.1",
			Port: 5001,
		},
		Server: &ServerConfig{
			Database: &DatabaseConfig{
				Path: filepath.Join(xdg.DataHome, "dorothy.db"),
			},
		},
	}

	return config.WriteConfig(w)
}
