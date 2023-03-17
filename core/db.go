package core

import (
	"os"
	"path/filepath"

	db "github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/sqlite"
)

func DorthyRoot() string {
	return os.Getenv("DORTHY_ROOT")
}

type DatabaseSession struct {
	db.Session
}

func NewDatabaseSession(config *Config) (*DatabaseSession, error) {
	database_path := filepath.Join(DorthyRoot(), "dorthy.db")

	settings := sqlite.ConnectionURL{
		Database: database_path,
	}

	s, err := sqlite.Open(settings)
	if err != nil {
		return nil, err
	}

	return &DatabaseSession{s}, nil
}

func (s *DatabaseSession) IsInitialized() bool {
	return false
}

func (s *DatabaseSession) Initialize() error {
	if s.IsInitialized() {
		return nil
	}

	return nil
}
