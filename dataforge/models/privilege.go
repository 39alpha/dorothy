package models

type RoleCode string

const (
	NoRole    RoleCode = "none"
	UserRole  RoleCode = "user"
	AdminRole RoleCode = "admin"
)

type Role struct {
	Code        RoleCode `json:"code" gorm:"primaryKey"`
	Description string   `json:"description"`
	Users       []User   `json:"users"`
}

type PrivilegeCode string

const (
	NoPrivilege    PrivilegeCode = "none"
	ReadPrivilege  PrivilegeCode = "read"
	WritePrivilege PrivilegeCode = "write"
	AdminPrivilege PrivilegeCode = "admin"
)

type Privilege struct {
	Code        PrivilegeCode `json:"code" gorm:"primaryKey"`
	Description string        `json:"description"`
}

type UserTeamPrivilege struct {
	UserID        uint          `json:"userId" gorm:"primaryKey"`
	TeamID        uint          `json:"TeamID" gorm:"primaryKey"`
	PrivilegeCode PrivilegeCode `json:"privilegeCode"`

	User      *User      `json:"user"`
	Team      *Team      `json:"Team"`
	Privilege *Privilege `json:"privilege"`
}

type UserDatasetPrivilege struct {
	UserID        uint          `json:"userId" gorm:"primaryKey"`
	DatasetID     uint          `json:"datasetId" gorm:"primaryKey"`
	PrivilegeCode PrivilegeCode `json:"privilegeCode"`

	User      *User      `json:"user"`
	Dataset   *Dataset   `json:"dataset"`
	Privilege *Privilege `json:"privilege"`
}
