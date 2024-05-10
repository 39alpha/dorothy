package cli

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
)

func SetUserName(name string) error {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	if config.User == nil {
		config.User = &core.UserConfig{
			Name: name,
		}
	} else {
		config.User.Name = name
	}

	return config.WriteFile(CONFIG_PATH)
}

func SetUserEmail(email string) error {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	if config.User == nil {
		config.User = &core.UserConfig{
			Email: email,
		}
	} else {
		config.User.Email = email
	}

	return config.WriteFile(CONFIG_PATH)
}

func SetRemote(remote string) error {
	r, err := core.NewRemote(remote)
	if err != nil {
		return err
	}

	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to open configuration file for reading")
	}

	config.RemoteString = r.String()

	return config.WriteFile(CONFIG_PATH)
}

func SetEditor(editor string) error {
	config, err := core.ReadConfigFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("failed to read configuration")
	}

	config.Editor = editor

	return config.WriteFile(CONFIG_PATH)
}
