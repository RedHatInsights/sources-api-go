package util

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func Capitalize(str string) string {
	caser := cases.Title(language.English)
	return caser.String(str)
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
