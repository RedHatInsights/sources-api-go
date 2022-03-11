package main

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v4"
)

var binder echo.Binder = &NoUnknownFieldsBinder{}

type TestStruct struct {
	YesIamAField bool `json:"good_field"`
}

func TestGoodPayload(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"good_field": true}`),
		nil,
	)
	// set the binder to our custom instance.
	c.Echo().Binder = binder

	err := c.Bind(&TestStruct{})
	if err != nil {
		t.Error(err)
	}

	// resetting due to messing with other tests.
	c.Echo().Binder = &echo.DefaultBinder{}
}

func TestBadPayload(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"good_field": true, "oops": "yes"}`),
		nil,
	)
	c.Echo().Binder = binder

	err := c.Bind(&TestStruct{})
	if err == nil {
		t.Error("No error was found when there should have been an extra field error")
	}

	c.Echo().Binder = &echo.DefaultBinder{}
}

func TestNilBody(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		nil,
	)
	c.Echo().Binder = binder

	err := c.Bind(&TestStruct{})
	if err == nil {
		t.Error("No error was found when there should have been a no body error")
	}

	c.Echo().Binder = &echo.DefaultBinder{}
}

func TestNoBody(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		http.NoBody,
		nil,
	)
	c.Echo().Binder = binder

	err := c.Bind(&TestStruct{})
	if err == nil {
		t.Error("No error was found when there should have been a no body error")
	}

	c.Echo().Binder = &echo.DefaultBinder{}
}
