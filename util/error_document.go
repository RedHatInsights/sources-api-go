package util

import (
	"fmt"
	"reflect"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

var ErrNotFoundEmpty = NewErrNotFound("")
var ErrBadRequestEmpty = NewErrBadRequest("")

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
	switch t := t.(type) {
	case string:
		return ErrBadRequest{Message: t}
	case error:
		return ErrBadRequest{Message: t.Error()}
	default:
		panic("bad interface type for bad request: " + reflect.ValueOf(t).String())
	}
}
