package util

import (
	"fmt"
	"reflect"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

var ErrNotFoundEmpty = NewErrNotFound("")
var ErrBadRequestEmpty = NewErrBadRequest("")
var callerDepth = 2

type Error struct {
	Detail string `json:"detail"`
	Status string `json:"status"`
}
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func ErrorDocWithoutLogging(message, status string) *ErrorDocument {
	return ErrorDocRaw(message, status)
}

func ErrorDoc(message, status string) *ErrorDocument {
	l.Log.WithField("caller_depth", callerDepth).Error(message)

	return ErrorDocRaw(message, status)
}

func ErrorDocRaw(message, status string) *ErrorDocument {
	return &ErrorDocument{
		[]Error{{
			Detail: message,
			Status: status,
		}},
	}
}

type ErrNotFound struct {
	Type string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found", e.Type)
}

func (e ErrNotFound) Is(err error) bool {
	return reflect.TypeOf(err) == reflect.TypeOf(e)
}

func NewErrNotFound(t string) error {
	if l.Log != nil {
		l.Log.WithField("caller_depth", callerDepth).Error(t)
	}

	return ErrNotFound{Type: t}
}

type ErrBadRequest struct {
	Message string
}

func (e ErrBadRequest) Error() string {
	return fmt.Sprintf("bad request: %s", e.Message)
}

func (e ErrBadRequest) Is(err error) bool {
	return reflect.TypeOf(err) == reflect.TypeOf(e)
}

func NewErrBadRequest(t interface{}) error {
	errorMessage := ""

	switch t := t.(type) {
	case string:
		errorMessage = t
	case error:
		errorMessage = t.Error()
	default:
		panic("bad interface type for bad request: " + reflect.ValueOf(t).String())
	}

	if l.Log != nil {
		l.Log.WithField("caller_depth", callerDepth).Error(errorMessage)
	}

	return ErrBadRequest{Message: errorMessage}
}
