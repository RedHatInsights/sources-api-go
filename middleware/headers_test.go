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

// TestParseAll tests that all the headers are correctly parsed by the middleware. It also checks that when all three
// "ebs account number", "org id" and "x-rh-identity" headers are provided, the last one is favored when populating the
// identity struct.
func TestParseAll(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set(h.IdentityKey, xrhid)
	c.Request().Header.Set(h.PskKey, "test-psk")
	c.Request().Header.Set(h.AccountNumberKey, "test-ebs-account-number")
	c.Request().Header.Set(h.XrhUserIdKey, "test-psk-user")
	c.Request().Header.Set(h.OrgIdKey, "test-orgid")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PskKey).(string) != "test-psk" {
		t.Errorf("%v was set as psk instead of %v", c.Get(h.PskKey).(string), "test-psk")
	}

	if c.Get(h.AccountNumberKey).(string) != "test-ebs-account-number" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumberKey).(string), "test-ebs-account-number")
	}

	if c.Get(h.XrhUserIdKey).(string) != "test-psk-user" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.XrhUserIdKey).(string), "test-psk-user")
	}

	if c.Get(h.OrgIdKey).(string) != "test-orgid" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get(h.OrgIdKey).(string))
	}

	id, ok := c.Get(h.ParsedIdentityKey).(*identity.XRHID)
	if !ok {
		t.Errorf(`unexpected type of identity received. Want "*identity.XRHID", got "%s"`, reflect.TypeOf(c.Get(h.ParsedIdentityKey)))
	}

	if id.Identity.AccountNumber != "12345" {
		t.Errorf("%v was set as identity account-number instead of %v", id.Identity.AccountNumber, "12345")
	}

	if id.Identity.OrgID != "23456" {
		t.Errorf(`invalid OrgId extracted from the identity. Want "%s", got "%s"`, "23456", id.Identity.OrgID)
	}
}

// TestParseWithoutXrhid tests that when no "x-rh-identity" header is provided, the identity struct is generated from
// the "ebs account number" and "org id" headers.
func TestParseWithoutXrhid(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set(h.PskKey, "test-psk")
	c.Request().Header.Set(h.AccountNumberKey, "test-ebs-account-number")
	c.Request().Header.Set(h.XrhUserIdKey, "test-psk-user")
	c.Request().Header.Set(h.OrgIdKey, "test-orgid")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PskKey).(string) != "test-psk" {
		t.Errorf("%v was set as psk instead of %v", c.Get(h.PskKey).(string), "test-psk")
	}

	if c.Get(h.AccountNumberKey).(string) != "test-ebs-account-number" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumberKey).(string), "test-ebs-account-number")
	}

	if c.Get(h.XrhUserIdKey).(string) != "test-psk-user" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.XrhUserIdKey).(string), "test-psk-user")
	}

	if c.Get(h.OrgIdKey).(string) != "test-orgid" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get(h.OrgIdKey).(string))
	}

	id, ok := c.Get(h.ParsedIdentityKey).(*identity.XRHID)
	if !ok {
		t.Errorf(`unexpected type of identity received. Want "*identity.XRHID", got "%s"`, reflect.TypeOf(c.Get(h.ParsedIdentityKey)))
	}

	if id.Identity.AccountNumber != "test-ebs-account-number" {
		t.Errorf("%v was set as identity account-number instead of %v", id.Identity.AccountNumber, "test-ebs-account-number")
	}

	if id.Identity.OrgID != "test-orgid" {
		t.Errorf(`invalid OrgId extracted from the identity. Want "%s", got "%s"`, "23456", id.Identity.OrgID)
	}
}

func TestParseAccountNumber(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set(h.AccountNumberKey, "9876")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.AccountNumberKey).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumberKey).(string), "9876")
	}
}

func TestBadIdentityBase64(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Set(h.IdentityKey, "not valid base64")

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
	c.Request().Header.Set(h.IdentityKey, "eyJub3QgYSByZWFsIGZpZWxkIjogdHJ1ZX0gLW4K")

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

	c.Request().Header.Set(h.PskKey, "1234")
	c.Request().Header.Set(h.AccountNumberKey, "9876")
	c.Request().Header.Set(h.XrhUserIdKey, "555555")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PskKey).(string) != "1234" {
		t.Errorf("%v was set as psk instead of %v", c.Get("psk").(string), "1234")
	}

	if c.Get(h.AccountNumberKey).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumberKey).(string), "9876")
	}

	if c.Get(h.XrhUserIdKey).(string) != "555555" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.XrhUserIdKey).(string), "555555")
	}
}
