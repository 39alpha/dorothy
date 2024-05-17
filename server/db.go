package server

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func DorothyRoot() string {
	return os.Getenv("DORTHY_ROOT")
}

type DatabaseSession struct {
	*gorm.DB
}

func NewDatabaseSession(config *core.DatabaseConfig) (*DatabaseSession, error) {
	if config == nil {
		return nil, fmt.Errorf("no server database configuration provided")
	}

	path := config.Path + "?_foreign_keys=on&cache=shared"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &DatabaseSession{db}, nil
}

func (s *DatabaseSession) Initialize() error {
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
	if result := s.Save(&roles); result.Error != nil {
		return result.Error
	}

	privileges := []*model.Privilege{
		{Code: "read", Description: "Read access"},
		{Code: "write", Description: "Write access"},
		{Code: "admin", Description: "Administrative access"},
	}
	if result := s.Save(&privileges); result.Error != nil {
		return result.Error
	}

	return nil
}

func (s *DatabaseSession) CreateUser(newuser *model.NewUser) error {
	var result struct {
		Count int
	}
	err := s.Raw("SELECT COUNT(*) AS count FROM users").First(&result).Error
	if err != nil {
		return fmt.Errorf("failed to get user count")
	}

	rolecode := "user"
	if result.Count == 0 {
		rolecode = "admin"
	}

	password_hash, err := bcrypt.GenerateFromPassword([]byte(newuser.Password), 8)
	if err != nil {
		return fmt.Errorf("failed to create user")
	}

	user := &model.User{
		Email:        newuser.Email,
		PasswordHash: password_hash,
		Name:         newuser.Name,
		Orcid:        newuser.Orcid,
		RoleCode:     rolecode,
	}

	return s.Create(user).Error
}

func (s *DatabaseSession) ValidateCredentials(email, password string) error {
	user := model.User{
		Email: email,
	}

	err := s.Select("PasswordHash").Where("email = ?", user.Email).First(&user).Error
	if err != nil || user.PasswordHash == nil {
		return fmt.Errorf("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return fmt.Errorf("invalid email or password")
	}

	return nil
}
