package model

import "gorm.io/gorm"

type Dataset struct {
	gorm.Model
	ID             string       `json:"id" gorm:"primaryKey"`
	Name           string       `json:"name"`
	Contact        string       `json:"contact"`
	Description    string       `json:"description"`
	OrganizationID string       `json:"organizationId"`
	Organization   Organization `json:"organization"`
}

type NewDataset struct {
	Name           string  `json:"name"`
	Contact        string  `json:"contact"`
	Description    *string `json:"description,omitempty"`
	OrganizationID string  `json:"organizationId"`
}

func (input *NewDataset) ID() string {
	return Slugify(input.Name)
}

type GetDatasets struct {
	OrganizationID string `json:"organizationId"`
}

type GetDataset struct {
	GetDatasets
	ID string `json:"id"`
}
