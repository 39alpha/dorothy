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
	path := filepath.Join(DorothyRoot(), "dorothy.db")
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

	s.AutoMigrate(&model.Organization{})

	return nil
}
