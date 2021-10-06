package util

import l "github.com/RedHatInsights/sources-api-go/logger"

type Error struct {
	Detail string `json:"detail"`
	Status string `json:"status"`
}
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func ErrorDoc(message, status string) *ErrorDocument {
	l.Log.Error(message)

	return &ErrorDocument{
		[]Error{{
			Detail: message,
			Status: status,
		}},
	}
}
