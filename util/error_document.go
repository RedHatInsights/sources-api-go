package util

import (
	"fmt"
	"reflect"
)

type Error struct {
	Detail    string `json:"detail"`
	Status    string `json:"status"`
	RequestId string `json:"request_id,omitempty"`
}
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func ErrorDocWithRequestId(message, status, uuid string) *ErrorDocument {
	e := NewErrorDoc(message, status)
	e.Errors[0].RequestId = uuid

	return e
}

func NewErrorDoc(message, status string) *ErrorDocument {
	return &ErrorDocument{
		[]Error{
			{
				Detail: message,
				Status: status,
			},
		},
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

	return ErrBadRequest{Message: errorMessage}
}
