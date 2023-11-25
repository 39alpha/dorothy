package core

import (
	"context"

	"github.com/39alpha/dorothy/core/model"
)

func CreateOrganization(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewOrganization) (*model.Organization, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	org := &model.Organization{
		Id:          input.Id(),
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
	result := db.Where("Id = ?", input.Id).First(&organization)
	if result.Error != nil {
		return nil, result.Error
	}
	return &organization, nil
}
