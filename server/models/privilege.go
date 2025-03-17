package models

type Role struct {
	Code        string `json:"code" gorm:"primaryKey"`
	Description string `json:"description"`
	Users       []User `json:"users"`
}

type Privilege struct {
	Code        string `json:"code" gorm:"primaryKey"`
	Description string `json:"description"`
}

type UserTeamPrivilege struct {
	UserID        uint   `json:"userId" gorm:"primaryKey"`
	TeamID        uint   `json:"TeamID" gorm:"primaryKey"`
	PrivilegeCode string `json:"privilegeCode"`

	User      *User      `json:"user"`
	Team      *Team      `json:"Team"`
	Privilege *Privilege `json:"privilege"`
}

type UserDatasetPrivilege struct {
	UserID        uint   `json:"userId" gorm:"primaryKey"`
	DatasetID     uint   `json:"datasetId" gorm:"primaryKey"`
	PrivilegeCode string `json:"privilegeCode"`

	User      *User      `json:"user"`
	Dataset   *Dataset   `json:"dataset"`
	Privilege *Privilege `json:"privilege"`
}
