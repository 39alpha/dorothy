package db

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/log"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
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

	logger, _ := log.NewGormLogger(config.Log)
	path := config.Path + "?_foreign_keys=on&cache=shared"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger,
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
		&models.PasswordReset{},
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

func (d *DB) NewUser(newuser *models.NewUser) error {
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

func GenerateRandomPassword() (string, error) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func (d *DB) CreateUser(create models.CreateUser) error {
	var result struct {
		Count int
	}
	err := d.Raw("SELECT COUNT(*) AS count FROM users").First(&result).Error
	if err != nil {
		return fmt.Errorf("failed to get user count")
	}

	password, err := GenerateRandomPassword()
	if err != nil {
		return fmt.Errorf("failed to create user")
	}

	password_hash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return fmt.Errorf("failed to create user")
	}

	user := &models.User{
		Email:        create.Email,
		PasswordHash: password_hash,
		Name:         create.Name,
		Orcid:        create.Orcid,
		RoleCode:     create.Role,
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

func (d *DB) ValidateResetCredentials(token, resetPassword string, delete bool) (*models.User, error) {
	reset := &models.PasswordReset{ID: token}
	if err := d.Model(reset).First(reset).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(reset.ResetHash, []byte(resetPassword)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if time.Now().Sub(reset.CreatedAt) > 24*time.Hour {
		return nil, fmt.Errorf("reset has expired")
	}

	user := &models.User{ID: reset.UserID}
	if err := d.Omit("PasswordHash").Find(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if delete {
		if err := d.Delete(reset).Error; err != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
	}

	return user, nil
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
			Preload("Team").
			Select("datasets.*").
			Joins("INNER JOIN `teams` ON `teams`.`id` = `datasets`.`team_id`").
			Where("`teams`.`is_private` = 0 AND `datasets`.`is_private` = 0").
			Order("`datasets`.`updated_at` desc").
			Limit(6).
			Find(&datasets).
			Error
	}

	return datasets, db.Model(&models.Dataset{}).
		Preload("Team").
		Select("datasets.*").
		Joins("LEFT JOIN `user_dataset_privileges` AS `udp` ON `udp`.`dataset_id` = `datasets`.`id` AND `udp`.`user_id` = ?", user.ID).
		Joins("INNER JOIN `teams` ON `teams`.`id` = `datasets`.`team_id`").
		Joins("LEFT JOIN `user_team_privileges` AS `utp` ON `utp`.`team_id` = `teams`.`id` AND `utp`.`user_id` = ?", user.ID).
		Where("`udp`.`privilege_code` NOT IN ('', ?) OR (`utp`.`privilege_code` NOT IN ('', ?) AND `datasets`.`is_private` = 0)",
			models.NoPrivilege,
			models.NoPrivilege,
		).
		Order("`datasets`.`updated_at` desc").
		Limit(6).
		Find(&datasets).
		Error
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

func (db *DB) UpdateUserWithRole(update models.UpdateUserWithRole) error {
	values := map[string]any{
		"Name":     update.Name,
		"Email":    update.Email,
		"Orcid":    update.Orcid,
		"RoleCode": update.Role,
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

func (d *DB) DeleteUser(user *models.User) error {
	return d.Delete(user).Error
}

func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	user := models.User{Email: email}
	if err := db.Omit("PasswordHash").Where(user).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) CreatePasswordReset(email string) (string, error) {
	user, err := db.GetUserByEmail(email)
	if err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	if err = db.Where("user_id = ?", user.ID).Delete(&models.PasswordReset{}).Error; err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	password, err := GenerateRandomPassword()
	if err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	resetHash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	token, err := GenerateRandomPassword()
	if err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	reset := models.PasswordReset{
		ID:        token,
		UserID:    user.ID,
		ResetHash: resetHash,
	}

	if err = db.Create(&reset).Error; err != nil {
		return "", fmt.Errorf("failed to create password reset")
	}

	token = fmt.Sprintf("id=%s&token=%s", url.QueryEscape(password), url.QueryEscape(token))
	return token, nil
}

func (db *DB) InviteUser(create models.CreateUser) (string, error) {
	var token string
	return token, db.Transaction(func(tx *gorm.DB) error {
		if err := db.CreateUser(create); err != nil {
			return err
		}

		var err error
		token, err = db.CreatePasswordReset(create.Email)
		if err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
		}

		return nil
	})
}
