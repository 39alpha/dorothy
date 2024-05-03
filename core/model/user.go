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

func (user User) OrganizationPrivilege(org Organization) string {
	if user.RoleCode == "admin" {
		return "admin"
	}

	for _, privilege := range user.OrganizationPrivileges {
		if privilege.OrganizationID == org.ID {
			return privilege.PrivilegeCode
		}
	}

	if org.IsPrivate {
		return ""
	} else {
		return "read"
	}
}

func (user User) CanReadOrganization(org Organization) bool {
	return user.OrganizationPrivilege(org) != ""
}

func (user User) CanWriteOrganization(org Organization) bool {
	privilege := user.OrganizationPrivilege(org)
	return privilege != "" && privilege != "read"
}

func (user User) CanManageOrganization(org Organization) bool {
	return user.OrganizationPrivilege(org) == "admin"
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

	org := dataset.Organization
	if dataset.IsPrivate {
		if user.CanManageOrganization(*org) {
			return "admin"
		}
	} else if user.CanReadOrganization(*org) {
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
