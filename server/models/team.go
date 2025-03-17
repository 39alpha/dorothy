package models

import (
	"time"

	"gorm.io/gorm"
)

type Team struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Slug        string         `json:"slug" gorm:"uniqueIndex"`
	Name        string         `json:"name"`
	Contact     string         `json:"contact" gorm:"index"`
	Description string         `json:"description"`
	IsPrivate   bool           `json:"private"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	Datasets       []Dataset           `json:"datasets"`
	UserPrivileges []UserTeamPrivilege `json:"userPrivileges"`
}

type NewTeam struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Contact     string  `json:"contact"`
	Description *string `json:"description,omitempty"`
	IsPrivate   bool    `json:"private"`
}

type GetTeam struct {
	ID   *uint   `json:"id,omitempty"`
	Slug *string `json:"slug,omitempty"`
}

type UpdateTeam struct {
	NewTeam
	ID uint `json:"id"`
}
