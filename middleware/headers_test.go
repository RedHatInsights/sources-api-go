package middleware

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var parseOrElse204 = ParseHeaders(func(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
})

func TestParseAll(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set("x-rh-identity", xrhid)
	c.Request().Header.Set("x-rh-sources-psk", "1234")
	c.Request().Header.Set("x-rh-sources-org-id", "abcde")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get("psk").(string) != "1234" {
		t.Errorf("%v was set as psk instead of %v", c.Get("psk").(string), "1234")
	}

	// Gets set from the xrhid's account number.
	if c.Get("psk-account").(string) != "12345" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get("psk-account").(string), "9876")
	}

	if c.Get("psk-org-id").(string) != "abcde" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get("x-rh-sources-org-id").(string))
	}

	id, ok := c.Get("identity").(*identity.XRHID)
	if !ok {
		t.Errorf(`unexpected type of identity received. Want "*identity.XRHID", got "%s"`, reflect.TypeOf(c.Get("identity")))
	}

	if id.Identity.AccountNumber != "12345" {
		t.Errorf("%v was set as identity account-number instead of %v", id.Identity.AccountNumber, "12345")
	}
}

func TestParseAccountNumber(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set("x-rh-sources-account-number", "9876")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get("psk-account").(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get("psk-account").(string), "9876")
	}
}

func TestBadIdentityBase64(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set("x-rh-identity", "not valid base64")

	err := parseOrElse204(c)
	if err == nil {
		t.Errorf("there was no error when there should have been one")
	}

	want := "error decoding Identity: illegal base64"
	if !strings.Contains(err.Error(), want) {
		t.Errorf(`unexpected error message. Want "%s", got "%s"`, want, err)
	}

	if rec.Code != 200 {
		t.Errorf("%v was returned instead of %v", rec.Code, 200)
	}
}

func TestBadIdentityJson(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	// base64 for: {"not a real field": true}
	c.Request().Header.Set("x-rh-identity", "eyJub3QgYSByZWFsIGZpZWxkIjogdHJ1ZX0gLW4K")

	err := parseOrElse204(c)
	if err == nil {
		t.Errorf("there was no error when there should have been one")
	}

	want := "x-rh-identity header does not contain valid JSON"
	if !strings.Contains(err.Error(), want) {
		t.Errorf(`unexpected error message. Want "%s", got "%s"`, want, err)
	}

	if rec.Code != 200 {
		t.Errorf("%v was returned instead of %v", rec.Code, 200)
	}
}

func TestOnlyPskHeaders(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set("x-rh-sources-psk", "1234")
	c.Request().Header.Set("x-rh-sources-account-number", "9876")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get("psk").(string) != "1234" {
		t.Errorf("%v was set as psk instead of %v", c.Get("psk").(string), "1234")
	}

	if c.Get("psk-account").(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get("psk-account").(string), "9876")
	}
}
