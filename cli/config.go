package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

type Remote struct {
	Host         string
	Organization string
	Dataset      string
}

func ParseRemote(remote string) (Remote, error) {
	r := Remote{}

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
	return "http://" + r.Host + "/v0/organization/" + r.Organization + "/dataset/" + r.Dataset
}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (u *User) String() string {
	s := u.Name
	if s != "" {
		s += " "
	}
	if u.Email != "" {
		s += "<" + u.Email + ">"
	}
	return s
}

type Config struct {
	Editor string `json:"editor,omitempty"`
	Remote string `json:"remote"`
	User   *User  `json:"user"`
}

func ReadConfigFile(filename string) (*Config, error) {
	handle, err := os.Open(filename)
	defer handle.Close()
	if err != nil {
		return nil, err
	}

	return ReadConfig(handle)
}

func ReadConfig(r io.Reader) (*Config, error) {
	var config Config
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&config)
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
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	return encoder.Encode(config)
}

func SetUserName(name string) error {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	if config.User == nil {
		config.User = &User{
			Name: name,
		}
	} else {
		config.User.Name = name
	}

	return config.WriteConfigFile(configpath)
}

func SetUserEmail(email string) error {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	if config.User == nil {
		config.User = &User{
			Email: email,
		}
	} else {
		config.User.Email = email
	}

	return config.WriteConfigFile(configpath)
}

func SetRemote(remote string) error {
	r, err := ParseRemote(remote)
	if err != nil {
		return err
	}

	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to open configuration file for reading")
	}

	config.Remote = r.String()

	return config.WriteConfigFile(configpath)
}

func SetEditor(editor string) error {
	config, err := ReadConfigFile(configpath)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	config.Editor = editor

	return config.WriteConfigFile(configpath)
}
