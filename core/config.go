package core

import (
	"path/filepath"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

type Config struct {
	Filename string     `toml:"-"`
	Ipfs     IpfsConfig `toml:"ipfs"`
	Database DatabaseConfig `toml:"database"`
}

type IpfsConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

func ReadConfig(filename string) (*Config, error) {
	config := Config{
		Filename: filename,
	}

	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func GenerateConfig(w io.Writer) error {
	config := Config{
		Filename: "",
		Ipfs: IpfsConfig{
			Host: "127.0.0.1",
			Port: 5001,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(xdg.DataHome, "dorothy.db"),
		},
	}

	encoder := toml.NewEncoder(w)
	encoder.Encode(&config)

	return nil
}

func (c IpfsConfig) Url() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
