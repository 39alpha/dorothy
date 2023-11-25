package model

import (
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var invalidChar *regexp.Regexp
var spaces *regexp.Regexp

func init() {
	invalidChar = regexp.MustCompile(`[^a-z\- ]`)
	spaces = regexp.MustCompile(`[[:space:]]+`)
}

type Organization struct {
	gorm.Model
	Id          string `json:"id" gorm:"primaryKey"`
	Name        string `json:"name"`
	Contact     string `json:"contact" gorm:"index"`
	Description string `json:"description"`
}

type NewOrganization struct {
	Name        string  `json:"name"`
	Contact     string  `json:"contact"`
	Description *string `json:"description,omitempty"`
}

func (input NewOrganization) Id() string {
	id := strings.TrimSpace(input.Name)
	id = strings.ToLower(id)
	id = invalidChar.ReplaceAllString(id, "")
	return spaces.ReplaceAllString(id, "-")
}
