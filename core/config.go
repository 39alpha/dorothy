package core

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Filename string     `toml:"-"`
	Ipfs     IpfsConfig `toml:"ipfs"`
}

type IpfsConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
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
	}

	encoder := toml.NewEncoder(w)
	encoder.Encode(&config)

	return nil
}

func (c IpfsConfig) Url() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
