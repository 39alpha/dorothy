package db

import (
	"fmt"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

func Open(config *core.DatabaseConfig) (*DB, error) {
	if config == nil {
		return nil, fmt.Errorf("no server database configuration provided")
	}

	path := config.Path + "?_foreign_keys=on&cache=shared"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (d *DB) Initialize() error {
	err := d.AutoMigrate(
		&models.Role{},
		&models.Privilege{},
		&models.Team{},
		&models.Dataset{},
		&models.User{},
		&models.UserTeamPrivilege{},
		&models.UserDatasetPrivilege{},
	)
	if err != nil {
		return err
	}

	roles := []*models.Role{
		{Code: "admin", Description: "The all-powerful entity"},
		{Code: "user", Description: "A standard user"},
	}
	if result := d.Save(&roles); result.Error != nil {
		return result.Error
	}

	privileges := []*models.Privilege{
		{Code: "read", Description: "Read access"},
		{Code: "write", Description: "Write access"},
		{Code: "admin", Description: "Administrative access"},
	}
	if result := d.Save(&privileges); result.Error != nil {
		return result.Error
	}

	return nil
}

func (d *DB) CreateUser(newuser *models.NewUser) error {
	var result struct {
		Count int
	}
	err := d.Raw("SELECT COUNT(*) AS count FROM users").First(&result).Error
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

	return d.Create(user).Error
}

func (d *DB) ValidateCredentials(email, password string) error {
	user := models.User{
		Email: email,
	}

	err := d.Select("PasswordHash").Where("email = ?", user.Email).First(&user).Error
	if err != nil || user.PasswordHash == nil {
		return fmt.Errorf("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return fmt.Errorf("invalid email or password")
	}

	return nil
}

func (d *DB) CreateDataset(newdata models.NewDataset, manifest *core.Manifest, user *models.User) (string, error) {
	dataset := &models.Dataset{
		Name:         models.Slugify(newdata.Name),
		TeamID:       newdata.TeamID,
		Contact:      newdata.Contact,
		IsPrivate:    newdata.IsPrivate,
		ManifestHash: manifest.Hash,
	}
	if newdata.Description != nil {
		dataset.Description = *newdata.Description
	}

	return dataset.Name, d.Transaction(func(tx *gorm.DB) error {
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

func (d *DB) GetTeams(user *models.User, loaddatasets bool) ([]models.Team, error) {
	allteams := []models.Team{}
	if loaddatasets {
		if err := d.Preload("Datasets.Team").Find(&allteams).Error; err != nil {
			return nil, err
		}
	} else {
		if err := d.Find(&allteams).Error; err != nil {
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

func (db *DB) GetTeamById(authUser *models.User, id uint) (*models.Team, error) {
	team := &models.Team{ID: id}
	if err := db.Preload("Datasets.Team").Where(team).First(team).Error; err != nil {
		return nil, err
	}

	datasets := []models.Dataset{}
	if authUser == nil {
		for _, dataset := range team.Datasets {
			if !dataset.IsPrivate {
				datasets = append(datasets, dataset)
			}
		}
	} else {
		for _, dataset := range team.Datasets {
			if authUser.CanReadDataset(dataset) {
				datasets = append(datasets, dataset)
			}
		}
	}

	team.Datasets = datasets

	return team, nil
}

func (db *DB) GetTeam(authUser *models.User, name string) (*models.Team, error) {
	if name == "" {
		return nil, gorm.ErrRecordNotFound
	}

	team := &models.Team{Name: name}
	if err := db.Preload("Datasets.Team").Where(team).First(team).Error; err != nil {
		return nil, err
	}

	datasets := []models.Dataset{}
	if authUser == nil {
		for _, dataset := range team.Datasets {
			if !dataset.IsPrivate {
				datasets = append(datasets, dataset)
			}
		}
	} else {
		for _, dataset := range team.Datasets {
			if authUser.CanReadDataset(dataset) {
				datasets = append(datasets, dataset)
			}
		}
	}

	team.Datasets = datasets

	return team, nil
}

func (db *DB) CreateTeam(newteam models.NewTeam, user *models.User) (string, error) {
	team := &models.Team{
		Name:      models.Slugify(newteam.Name),
		Contact:   newteam.Contact,
		IsPrivate: newteam.IsPrivate,
	}
	if newteam.Description != nil {
		team.Description = *newteam.Description
	}

	return team.Name, db.Transaction(func(tx *gorm.DB) error {
		if err := db.Save(team).Error; err != nil {
			return err
		}

		if user != nil {
			return db.Save(&models.UserTeamPrivilege{
				User:          user,
				Team:          team,
				PrivilegeCode: "admin",
			}).Error
		}

		return nil
	})
}

func (d *DB) UpdateTeam(update models.UpdateTeam) (string, error) {
	name := models.Slugify(update.Name)
	values := map[string]any{
		"Name":        name,
		"Contact":     update.Contact,
		"IsPrivate":   update.IsPrivate,
		"Description": "",
	}
	if update.Description != nil {
		values["Description"] = *update.Description
	}

	return name, d.Model(models.Team{ID: update.ID}).Updates(values).Error
}

func (d *DB) DeleteTeam(team *models.Team) error {
	return d.Delete(team).Error
}

func (db *DB) IsTeamNameAvailable(name string) (bool, error) {
	if name == "" {
		return false, gorm.ErrInvalidValue
	}

	var count int64
	team := &models.Team{Name: name}
	if err := db.Model(team).Where(team).Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

func (d *DB) GetDatasets(user *models.User) ([]models.Dataset, error) {
	alldatasets := []models.Dataset{}
	if err := d.Preload("Team").Find(&alldatasets).Error; err != nil {
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

func (db *DB) GetDatasetById(authUser *models.User, id uint) (*models.Dataset, error) {
	dataset := models.Dataset{ID: id}
	if err := db.Preload("Team").Where(&dataset).First(&dataset).Error; err != nil {
		return nil, err
	}

	return &dataset, nil
}

func (db *DB) GetDataset(authUser *models.User, teamName string, datasetName string) (*models.Dataset, error) {
	if datasetName == "" {
		return nil, gorm.ErrRecordNotFound
	}

	team, err := db.GetTeam(authUser, teamName)
	if err != nil {
		return nil, err
	}

	dataset := &models.Dataset{
		Name:   datasetName,
		TeamID: team.ID,
	}
	if err := db.Preload("Team").Where(dataset).First(dataset).Error; err != nil {
		return nil, err
	}

	return dataset, nil
}

func (d *DB) UpdateDataset(update models.UpdateDataset) (string, error) {
	name := models.Slugify(update.Name)
	values := map[string]any{
		"ID":          update.ID,
		"Name":        name,
		"TeamID":      update.TeamID,
		"Contact":     update.Contact,
		"IsPrivate":   update.IsPrivate,
		"Description": "",
	}
	if update.Description != nil {
		values["Description"] = *update.Description
	}

	return name, d.Model(models.Dataset{ID: update.ID}).Omit("ManifestHash").Updates(values).Error
}

func (d *DB) DeleteDataset(dataset *models.Dataset) error {
	return d.Delete(dataset).Error
}

func (db *DB) IsDatasetNameAvailable(team, name string) (bool, error) {
	if name == "" {
		return false, gorm.ErrInvalidValue
	}

	var count int64 = -1
	dataset := &models.Dataset{Name: name}

	err := db.Model(dataset).
		Joins("join teams on teams.id = datasets.team_id").
		Where("teams.name = ?", team).
		Where(dataset).
		Count(&count).
		Error

	return count == 0, err
}
