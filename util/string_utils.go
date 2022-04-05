package util

import "strings"

func Capitalize(str string) string {
	return strings.ToUpper(string(str[0])) + str[1:]
}

func StringRef(str string) *string {
	return &str
}
