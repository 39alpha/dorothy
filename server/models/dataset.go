package models

import (
	"time"

	"github.com/39alpha/dorothy/core"
)

type Dataset struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"uniqueIndex:dataset"`
	Contact      string         `json:"contact"`
	Description  string         `json:"description"`
	IsPrivate    bool           `json:"private"`
	TeamID       uint           `json:"teamId" gorm:"uniqueIndex:dataset"`
	ManifestHash string         `json:"manifestHash"`
	Manifest     *core.Manifest `json:"manifest" gorm:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`

	Team           *Team                  `json:"team"`
	UserPrivileges []UserDatasetPrivilege `json:"userPrivileges" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type NewDataset struct {
	Name        string  `json:"name"`
	TeamID      uint    `json:"teamId"`
	Contact     string  `json:"contact"`
	Description *string `json:"description,omitempty"`
	IsPrivate   bool    `json:"isPrivate"`
}

type GetDatasets struct {
	TeamID uint `json:"teamId"`
}

type GetDataset struct {
	GetDatasets
	ID uint `json:"id"`
}

type UpdateDataset struct {
	NewDataset
	ID uint `json:"id"`
}
