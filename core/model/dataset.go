package model

import (
	"time"

	"gorm.io/gorm"
)

type Dataset struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Slug           string         `json:"slug" gorm:"uniqueIndex"`
	Name           string         `json:"name"`
	Contact        string         `json:"contact"`
	Description    string         `json:"description"`
	IsPrivate      bool           `json:"private"`
	OrganizationID uint           `json:"organizationId"`
	Manifest       *Manifest      `json:"manifest" gorm:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	Organization   *Organization          `json:"organization"`
	UserPrivileges []UserDatasetPrivilege `json:"userPrivileges"`
}

type NewDataset struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	Contact        string  `json:"contact"`
	Description    *string `json:"description,omitempty"`
	OrganizationID uint    `json:"organizationId"`
}

func (input *NewDataset) ID() string {
	return Slugify(input.Name)
}

type GetDatasets struct {
	OrganizationID uint `json:"organizationId"`
}

type GetDataset struct {
	GetDatasets
	ID uint `json:"id"`
}
