package model

import "gorm.io/gorm"

type Organization struct {
	gorm.Model
	ID          string `json:"id" gorm:"primaryKey"`
	Name        string `json:"name"`
	Contact     string `json:"contact" gorm:"index"`
	Description string `json:"description"`
}

type NewOrganization struct {
	Name        string  `json:"name"`
	Contact     string  `json:"contact"`
	Description *string `json:"description,omitempty"`
}

func (input *NewOrganization) ID() string {
	return Slugify(input.Name)
}

type GetOrganization struct {
	ID string `json:"id"`
}
