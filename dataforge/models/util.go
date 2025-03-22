package models

import (
	"regexp"
	"strings"
)

var invalidChar *regexp.Regexp
var spaces *regexp.Regexp

func init() {
	invalidChar = regexp.MustCompile(`[^0-9a-zA-Z\_\- ]`)
	spaces = regexp.MustCompile(`[[:space:]]+`)
}

func Slugify(name string) string {
	slug := strings.TrimSpace(name)
	slug = strings.ToLower(slug)
	slug = invalidChar.ReplaceAllString(slug, "")
	return spaces.ReplaceAllString(slug, "-")
}
