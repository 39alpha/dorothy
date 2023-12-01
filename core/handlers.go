package core

import (
	"context"
	"fmt"

	"github.com/39alpha/dorothy/core/model"
	"golang.org/x/crypto/bcrypt"
)

func CreateOrganization(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewOrganization) (*model.Organization, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	org := &model.Organization{
		ID:          input.ID(),
		Name:        input.Name,
		Contact:     input.Contact,
		Description: description,
	}

	if result := db.Create(org); result.Error != nil {
		return nil, result.Error
	}

	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateOrganization(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func ListOrganizations(ctx context.Context, config *Config, db *DatabaseSession) ([]*model.Organization, error) {
	var organizations []*model.Organization
	result := db.Find(&organizations)
	if result.Error != nil {
		return nil, result.Error
	}
	return organizations, nil
}

func GetOrganization(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetOrganization) (*model.Organization, error) {
	var organization model.Organization
	result := db.Where(&model.Organization{ID: input.ID}).First(&organization)
	if result.Error != nil {
		return nil, result.Error
	}
	return &organization, nil
}

func CreateDataset(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewDataset) (*model.Dataset, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	dataset := &model.Dataset{
		ID:             input.ID(),
		Name:           input.Name,
		Contact:        input.Contact,
		Description:    description,
		OrganizationID: input.OrganizationID,
	}

	if result := db.Create(dataset); result.Error != nil {
		return nil, result.Error
	}

	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateDataset(ctx, dataset)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func ListDatasets(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetDatasets) ([]*model.Dataset, error) {
	var datasets []*model.Dataset
	result := db.
		Where(&model.Dataset{OrganizationID: input.OrganizationID}).
		Find(&datasets)
	if result.Error != nil {
		return nil, result.Error
	}
	return datasets, nil
}

func GetDataset(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetDataset) (*model.Dataset, error) {
	var dataset model.Dataset
	result := db.Where(&model.Dataset{ID: input.ID}).First(&dataset)
	if result.Error != nil {
		return nil, result.Error
	}
	return &dataset, nil
}

func GetManifest(ctx context.Context, config *Config, input *model.Dataset) (*model.Manifest, error) {
	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	manifest, err := client.GetManifest(ctx, input.OrganizationID, input.ID)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func GetUsers(ctx context.Context, config *Config, db *DatabaseSession) ([]*model.User, error) {
	var users []*model.User
	result := db.Select("email", "name", "orcid").Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

func GetUser(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetUser) (*model.User, error) {
	var user model.User
	result := db.
		Where(&model.User{Email: input.Email}).
		Find(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func CreateUser(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewUser) (*model.User, error) {
	password_hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 8)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        input.Email,
		PasswordHash: password_hash,
		Name:         input.Name,
		Orcid:        input.Orcid,
	}
	if result := db.Create(user); result.Error != nil {
		return nil, result.Error
	}
	user.PasswordHash = nil
	return user, nil
}

func ValidateCredentials(db *DatabaseSession, email, password string) error {
	user := &model.User{
		Email: email,
	}

	result := db.Select("PasswordHash").Where(user).Find(&user)
	if result.Error != nil || user.PasswordHash == nil {
		return fmt.Errorf("invalid email or password")
	}

	err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return fmt.Errorf("invalid email or password")
	}
	return nil
}
