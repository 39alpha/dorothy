package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Email        string         `json:"email" gorm:"uniqueIndex"`
	PasswordHash []byte         `json:"-"`
	Name         string         `json:"name"`
	Orcid        *string        `json:"orcid,omitempty" gorm:"index"`
	RoleCode     string         `json:"roleCode"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	Role                   *Role                       `json:"role"`
	OrganizationPrivileges []UserOrganizationPrivilege `json:"organizationPrivileges"`
	DatasetPrivileges      []UserDatasetPrivilege      `json:"datasetPrivileges"`
}

type GetUser struct {
	Email string `json:"email"`
}

type NewUser struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     string  `json:"name"`
	Orcid    *string `json:"orcid,omitempty"`
}

type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
