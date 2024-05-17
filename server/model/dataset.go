package model

import (
	"time"

	"github.com/39alpha/dorothy/core"
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
	ManifestHash   string         `json:"manifestHash"`
	Manifest       *core.Manifest `json:"manifest" gorm:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	Organization   *Organization          `json:"organization"`
	UserPrivileges []UserDatasetPrivilege `json:"userPrivileges"`
}

type NewDataset struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	OrganizationID uint    `json:"organizationId"`
	Contact        string  `json:"contact"`
	Description    *string `json:"description,omitempty"`
	IsPrivate      bool    `json:"isPrivate"`
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
