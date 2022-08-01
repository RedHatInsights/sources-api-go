package middleware

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
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

	c.Request().Header.Set(h.XRHID, xrhid)
	c.Request().Header.Set(h.PSK, "1234")
	c.Request().Header.Set(h.ACCOUNT_NUMBER, emptyIdentity.Identity.AccountNumber)
	c.Request().Header.Set(h.PSK_USER, "55555")
	c.Request().Header.Set(h.ORGID, emptyIdentity.Identity.OrgID)

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PSK).(string) != "1234" {
		t.Errorf("%v was set as psk instead of %v", c.Get(h.PSK).(string), "1234")
	}

	if c.Get(h.ACCOUNT_NUMBER).(string) != "12345" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.ACCOUNT_NUMBER).(string), "12345")
	}

	if c.Get(h.PSK_USER).(string) != "55555" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.PSK_USER).(string), "55555")
	}

	if c.Get(h.ORGID).(string) != "23456" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get(h.ORGID).(string))
	}

	id, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
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

	c.Request().Header.Set(h.ACCOUNT_NUMBER, "9876")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.ACCOUNT_NUMBER).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.ACCOUNT_NUMBER).(string), "9876")
	}
}

func TestBadIdentityBase64(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set(h.XRHID, "not valid base64")

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
	c.Request().Header.Set(h.XRHID, "eyJub3QgYSByZWFsIGZpZWxkIjogdHJ1ZX0gLW4K")

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

	c.Request().Header.Set(h.PSK, "1234")
	c.Request().Header.Set(h.ACCOUNT_NUMBER, "9876")
	c.Request().Header.Set(h.PSK_USER, "555555")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PSK).(string) != "1234" {
		t.Errorf("%v was set as psk instead of %v", c.Get("psk").(string), "1234")
	}

	if c.Get(h.ACCOUNT_NUMBER).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.ACCOUNT_NUMBER).(string), "9876")
	}

	if c.Get(h.PSK_USER).(string) != "555555" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.PSK_USER).(string), "555555")
	}
}
