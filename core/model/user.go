package model

import "gorm.io/gorm"

type User struct {
	gorm.Model

	Email        string  `json:"email" gorm:"uniqueIndex"`
	PasswordHash []byte  `json:"-"`
	Name         string  `json:"name"`
	Orcid        *string `json:"orcid,omitempty" gorm:"uniqueIndex"`
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
