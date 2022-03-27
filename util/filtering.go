package util

import "regexp"

// Simple regex - which matches only the characters.
// the filters would come in like this:
//
// filter[name][eq]
// or
// filter[source_type][name][eq]
//
// and get matched as:
//
// ["filter", "name", "eq"]
// or
// ["filter", "source_type", "name", "eq"]
var FilterRegex = regexp.MustCompile(`\w+`)

type Filter struct {
	Subresource string
	Name        string
	Operation   string
	Value       []string
}
