package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func Open(config *core.DatabaseConfig) (*DB, error) {
	if config == nil {
		return nil, fmt.Errorf("no server database configuration provided")
	}

	newlogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             50 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	path := config.Path + "?_foreign_keys=on&cache=shared"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: newlogger,
	})
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
		{Code: models.AdminRole, Description: "The all-powerful entity"},
		{Code: models.UserRole, Description: "A standard user"},
	}
	if result := d.Save(&roles); result.Error != nil {
		return result.Error
	}

	privileges := []*models.Privilege{
		{Code: models.ReadPrivilege, Description: "Read access"},
		{Code: models.WritePrivilege, Description: "Write access"},
		{Code: models.AdminPrivilege, Description: "Administrative access"},
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

	rolecode := models.UserRole
	if result.Count == 0 {
		rolecode = models.AdminRole
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

func (db *DB) GetHotTeams(user *models.User) ([]models.Team, error) {
	teams := []models.Team{}
	if user == nil {
		return teams, db.Order("updated_at desc").Limit(6).Find(&teams, "is_private = 0").Error
	} else {
		return teams, db.Model(&models.Team{}).
			Select("teams.*").
			Joins("INNER JOIN `user_team_privileges` ON `team_id` = `teams`.`id`").
			Where("`user_id` = ? AND `privilege_code` != ?", user.ID, models.NoPrivilege).
			Order("`teams`.`updated_at` desc").
			Limit(6).
			Find(&teams).
			Error
	}
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

func (db *DB) GetHotDatasets(user *models.User) ([]models.Dataset, error) {
	datasets := []models.Dataset{}
	if user == nil {
		return datasets, db.Model(&models.Dataset{}).
			Select("datasets.*").
			Joins("INNER JOIN `teams` ON `teams`.`id` = `datasets`.`team_id`").
			Where("`teams`.`is_private` = 0").
			Order("`datasets`.`updated_at` desc").
			Limit(6).
			Find(&datasets, "is_private = 0").
			Error
	} else {
		return datasets, db.Model(&models.Dataset{}).
			Preload("Team").
			Select("datasets.*").
			Joins("INNER JOIN `user_dataset_privileges` ON `dataset_id` = `datasets`.`id`").
			Where("`user_id` = ? AND `privilege_code` != ?", user.ID, models.NoPrivilege).
			Order("`datasets`.`updated_at` desc").
			Limit(6).
			Find(&datasets).
			Error
	}
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

func (db *DB) GetUsersWithTeamAccess(team models.Team) ([]models.User, error) {
	users := []models.User{}
	return users, db.
		Preload("TeamPrivileges", "team_id = ?", team.ID).
		Preload("TeamPrivileges.Privilege").
		Omit("PasswordHash").
		Find(&users).
		Error
}

func (db *DB) GetUsersWithDatasetAccess(dataset models.Dataset) ([]models.User, error) {
	users := []models.User{}
	return users, db.
		Preload("DatasetPrivileges", "dataset_id = ?", dataset.ID).
		Preload("DatasetPrivileges.Privilege").
		Omit("PasswordHash").
		Find(&users).
		Error
}

func (db *DB) SearchUsers(pattern string, limit int) ([]models.User, error) {
	pattern = fmt.Sprintf("%%%s%%", pattern)
	users := []models.User{}

	var err error
	if limit >= 0 {
		err = db.
			Omit("PasswordHash").
			Limit(limit).
			Find(&users, "name LIKE ? OR email LIKE ? or orcid LIKE ?", pattern, pattern, pattern).
			Error
	} else {
		err = db.
			Omit("PasswordHash").
			Find(&users, "name LIKE ? OR email LIKE ? or orcid LIKE ?", pattern, pattern, pattern).
			Error
	}

	return users, err
}

func (db *DB) UpdateDatasetPrivilege(privilege models.UserDatasetPrivilege) error {
	return db.Save(&privilege).Error
}

func (db *DB) DeleteDatasetPrivilege(privilege models.UserDatasetPrivilege) error {
	return db.Delete(&privilege).Error
}

func (db *DB) UpdateTeamPrivilege(privilege models.UserTeamPrivilege) error {
	return db.Save(&privilege).Error
}

func (db *DB) DeleteTeamPrivilege(privilege models.UserTeamPrivilege) error {
	return db.Delete(&privilege).Error
}

func (db *DB) GetUserById(id uint) (*models.User, error) {
	user := &models.User{ID: id}
	if err := db.Model(user).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) UpdateUser(update models.UpdateUser) error {
	values := map[string]any{
		"Name":  update.Name,
		"Email": update.Email,
		"Orcid": update.Orcid,
	}
	return db.Model(models.User{ID: update.ID}).Updates(values).Error
}

func (db *DB) UserChangePassword(update models.ChangePassword) error {
	password_hash, err := bcrypt.GenerateFromPassword([]byte(update.Password), 8)
	if err != nil {
		return fmt.Errorf("failed to update password")
	}

	values := map[string]any{
		"PasswordHash": password_hash,
	}

	return db.Model(&models.User{ID: update.ID}).Updates(values).Error
}
