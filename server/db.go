package server

import (
	"fmt"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server/models"
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
		&models.Role{},
		&models.Privilege{},
		&models.Team{},
		&models.Dataset{},
		&models.User{},
		&models.UserTeamPrivilege{},
		&models.UserDatasetPrivilege{},
	)

	roles := []*models.Role{
		{Code: "admin", Description: "The all-powerful entity"},
		{Code: "user", Description: "A standard user"},
	}
	if result := s.Save(&roles); result.Error != nil {
		return result.Error
	}

	privileges := []*models.Privilege{
		{Code: "read", Description: "Read access"},
		{Code: "write", Description: "Write access"},
		{Code: "admin", Description: "Administrative access"},
	}
	if result := s.Save(&privileges); result.Error != nil {
		return result.Error
	}

	return nil
}

func (s *DatabaseSession) CreateUser(newuser *models.NewUser) error {
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

	user := &models.User{
		Email:        newuser.Email,
		PasswordHash: password_hash,
		Name:         newuser.Name,
		Orcid:        newuser.Orcid,
		RoleCode:     rolecode,
	}

	return s.Create(user).Error
}

func (s *DatabaseSession) ValidateCredentials(email, password string) error {
	user := models.User{
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

func (s *DatabaseSession) CreateDataset(newdata models.NewDataset, manifest *core.Manifest, user *models.User) error {
	return s.Transaction(func(tx *gorm.DB) error {
		dataset := &models.Dataset{
			Slug:         newdata.Slug,
			Name:         newdata.Name,
			TeamID:       newdata.TeamID,
			Contact:      newdata.Contact,
			IsPrivate:    newdata.IsPrivate,
			ManifestHash: manifest.Hash,
		}
		if newdata.Description != nil {
			dataset.Description = *newdata.Description
		}

		if err := tx.Save(dataset).Error; err != nil {
			return err
		}

		if user != nil {
			return tx.Save(&models.UserDatasetPrivilege{
				User:          user,
				Dataset:       dataset,
				PrivilegeCode: "admin",
			}).Error
		}

		return nil
	})
}

func (s *DatabaseSession) GetTeams(user *models.User, loaddatasets bool) ([]models.Team, error) {
	allteams := []models.Team{}
	if loaddatasets {
		if err := s.Preload("Datasets.Team").Find(&allteams).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.Find(&allteams).Error; err != nil {
			return nil, err
		}
	}

	teams := []models.Team{}

	if user == nil {
		for _, team := range allteams {
			if team.IsPrivate {
				continue
			}

			datasets := []models.Dataset{}
			for _, dataset := range team.Datasets {
				if !dataset.IsPrivate {
					datasets = append(datasets, dataset)
				}
			}
			team.Datasets = datasets

			teams = append(teams, team)
		}
	} else {
		for _, team := range allteams {
			if !user.CanReadTeam(team) {
				continue
			}

			datasets := []models.Dataset{}
			for _, dataset := range team.Datasets {
				if user.CanReadDataset(dataset) {
					datasets = append(datasets, dataset)
				}
			}
			team.Datasets = datasets

			teams = append(teams, team)
		}
	}

	return teams, nil
}

func (s *DatabaseSession) GetDatasets(user *models.User) ([]models.Dataset, error) {
	alldatasets := []models.Dataset{}
	if err := s.Preload("Team").Find(&alldatasets).Error; err != nil {
		return nil, err
	}

	datasets := []models.Dataset{}

	if user == nil {
		for _, dataset := range alldatasets {
			if dataset.IsPrivate || dataset.Team.IsPrivate {
				continue
			}
			datasets = append(datasets, dataset)
		}
	} else {
		for _, dataset := range alldatasets {
			if !user.CanReadDataset(dataset) {
				continue
			}
			datasets = append(datasets, dataset)
		}
	}

	return datasets, nil
}

func (s *DatabaseSession) UpdateDataset(update models.UpdateDataset) error {
	values := map[string]any{
		"ID":          update.ID,
		"Slug":        update.Slug,
		"Name":        update.Name,
		"TeamID":      update.TeamID,
		"Contact":     update.Contact,
		"IsPrivate":   update.IsPrivate,
		"Description": "",
	}
	if update.Description != nil {
		values["Description"] = *update.Description
	}

	result := s.Model(models.Dataset{ID: update.ID}).Omit("ManifestHash").Updates(values)
	return result.Error
}

func (s *DatabaseSession) DeleteDataset(dataset *models.Dataset) error {
	return s.Delete(dataset).Error
}

func (s *DatabaseSession) UpdateTeam(update models.UpdateTeam) error {
	values := map[string]any{
		"Slug":        update.Slug,
		"Name":        update.Name,
		"Contact":     update.Contact,
		"IsPrivate":   update.IsPrivate,
		"Description": "",
	}
	if update.Description != nil {
		values["Description"] = *update.Description
	}

	return s.Model(models.Team{ID: update.ID}).Updates(values).Error
}
