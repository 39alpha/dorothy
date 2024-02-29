package core

import (
	"os"
	"path/filepath"

	"github.com/39alpha/dorothy/core/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func DorothyRoot() string {
	return os.Getenv("DORTHY_ROOT")
}

type DatabaseSession struct {
	*gorm.DB
}

func NewDatabaseSession(config *Config) (*DatabaseSession, error) {
	path := filepath.Join(DorothyRoot(), "dorothy.db?_foreign_keys=on")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &DatabaseSession{db}, nil
}

func (s *DatabaseSession) IsInitialized() bool {
	return false
}

func (s *DatabaseSession) Initialize() error {
	if s.IsInitialized() {
		return nil
	}

	s.AutoMigrate(
		&model.Role{},
		&model.Privilege{},
		&model.Organization{},
		&model.Dataset{},
		&model.User{},
		&model.UserOrganizationPrivilege{},
		&model.UserDatasetPrivilege{},
	)

	roles := []*model.Role{
		{Code: "admin", Description: "The all-powerful entity"},
		{Code: "user", Description: "A standard user"},
	}
	if result := session.Create(&roles); result.Error != nil {
		return result.Error
	}

	privileges := []*model.Privilege{
		{Code: "read", Description: "Read access"},
		{Code: "write", Description: "Write access"},
		{Code: "admin", Description: "Administrative access"},
	}
	if result := session.Create(&privileges); result.Error != nil {
		return result.Error
	}
}
