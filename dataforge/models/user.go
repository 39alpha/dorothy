package models

import (
	"time"
)

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex"`
	PasswordHash []byte    `json:"-"`
	Name         string    `json:"name"`
	Orcid        *string   `json:"orcid,omitempty" gorm:"index"`
	RoleCode     RoleCode  `json:"roleCode"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	Role              *Role                  `json:"role"`
	TeamPrivileges    []UserTeamPrivilege    `json:"teamPrivileges" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	DatasetPrivileges []UserDatasetPrivilege `json:"datasetPrivileges" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (user User) TeamPrivilege(team Team) PrivilegeCode {
	if user.RoleCode == AdminRole {
		return AdminPrivilege
	}

	for _, privilege := range user.TeamPrivileges {
		if privilege.TeamID == team.ID {
			return privilege.PrivilegeCode
		}
	}

	if team.IsPrivate {
		return NoPrivilege
	} else {
		return ReadPrivilege
	}
}

func (user User) CanReadTeam(team Team) bool {
	return user.TeamPrivilege(team) != NoPrivilege
}

func (user User) CanWriteTeam(team Team) bool {
	privilege := user.TeamPrivilege(team)
	return privilege != NoPrivilege && privilege != ReadPrivilege
}

func (user User) CanManageTeam(team Team) bool {
	return user.TeamPrivilege(team) == AdminPrivilege
}

func (user User) DatasetPrivilege(dataset Dataset) PrivilegeCode {
	if user.RoleCode == AdminRole {
		return AdminPrivilege
	}

	for _, privilege := range user.DatasetPrivileges {
		if privilege.DatasetID == dataset.ID {
			return privilege.PrivilegeCode
		}
	}

	team := dataset.Team
	if dataset.IsPrivate {
		if user.CanManageTeam(*team) {
			return AdminPrivilege
		}
	} else if user.CanReadTeam(*team) {
		return ReadPrivilege
	}

	return NoPrivilege
}

func (user User) CanReadDataset(dataset Dataset) bool {
	return user.DatasetPrivilege(dataset) != NoPrivilege
}

func (user User) CanWriteDataset(dataset Dataset) bool {
	privilege := user.DatasetPrivilege(dataset)
	return privilege != NoPrivilege && privilege != ReadPrivilege
}

func (user User) CanManageDataset(dataset Dataset) bool {
	return user.DatasetPrivilege(dataset) == AdminPrivilege
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
