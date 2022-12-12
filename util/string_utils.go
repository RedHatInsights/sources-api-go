package util

import "strings"

func Capitalize(str string) string {
	return strings.Title(strings.ToLower(str))
}

func StringRef(str string) *string {
	return &str
}

func ValueOrBlank(strRef *string) string {
	if strRef == nil {
		return ""
	}

	return *strRef
}
