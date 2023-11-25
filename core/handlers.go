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
