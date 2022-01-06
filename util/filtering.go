package util

import "regexp"

var FilterRegex = regexp.MustCompile(`^filter\[(\w+)](\[\w*]|$)`)

type Filter struct {
	Name      string
	Operation string
	Value     []string
}
