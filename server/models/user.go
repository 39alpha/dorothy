package models

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

	Role              *Role                  `json:"role"`
	TeamPrivileges    []UserTeamPrivilege    `json:"teamPrivileges"`
	DatasetPrivileges []UserDatasetPrivilege `json:"datasetPrivileges"`
}

func (user User) TeamPrivilege(team Team) string {
	if user.RoleCode == "admin" {
		return "admin"
	}

	for _, privilege := range user.TeamPrivileges {
		if privilege.TeamID == team.ID {
			return privilege.PrivilegeCode
		}
	}

	if team.IsPrivate {
		return ""
	} else {
		return "read"
	}
}

func (user User) CanReadTeam(team Team) bool {
	return user.TeamPrivilege(team) != ""
}

func (user User) CanWriteTeam(team Team) bool {
	privilege := user.TeamPrivilege(team)
	return privilege != "" && privilege != "read"
}

func (user User) CanManageTeam(team Team) bool {
	return user.TeamPrivilege(team) == "admin"
}

func (user User) DatasetPrivilege(dataset Dataset) string {
	if user.RoleCode == "admin" {
		return "admin"
	}

	for _, privilege := range user.DatasetPrivileges {
		if privilege.DatasetID == dataset.ID {
			return privilege.PrivilegeCode
		}
	}

	team := dataset.Team
	if dataset.IsPrivate {
		if user.CanManageTeam(*team) {
			return "admin"
		}
	} else if user.CanReadTeam(*team) {
		return "read"
	}

	return ""
}

func (user User) CanReadDataset(dataset Dataset) bool {
	return user.DatasetPrivilege(dataset) != ""
}

func (user User) CanWriteDataset(dataset Dataset) bool {
	privilege := user.DatasetPrivilege(dataset)
	return privilege != "" && privilege != "read"
}

func (user User) CanManageDataset(dataset Dataset) bool {
	return user.DatasetPrivilege(dataset) == "admin"
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
