package model

import (
	"time"

	"github.com/39alpha/dorothy/core"
	"gorm.io/gorm"
)

type Dataset struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Slug         string         `json:"slug" gorm:"uniqueIndex:dataset"`
	Name         string         `json:"name"`
	Contact      string         `json:"contact"`
	Description  string         `json:"description"`
	IsPrivate    bool           `json:"private"`
	TeamID       uint           `json:"teamId" gorm:"uniqueIndex:dataset"`
	ManifestHash string         `json:"manifestHash"`
	Manifest     *core.Manifest `json:"manifest" gorm:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	Team           *Team                  `json:"team"`
	UserPrivileges []UserDatasetPrivilege `json:"userPrivileges"`
}

type NewDataset struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	TeamID      uint    `json:"teamId"`
	Contact     string  `json:"contact"`
	Description *string `json:"description,omitempty"`
	IsPrivate   bool    `json:"isPrivate"`
}

func (input *NewDataset) ID() string {
	return Slugify(input.Name)
}

type GetDatasets struct {
	TeamID uint `json:"teamId"`
}

type GetDataset struct {
	GetDatasets
	ID uint `json:"id"`
}
