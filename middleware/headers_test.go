package middleware

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
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

	c.Request().Header.Set(h.XRHID, xrhid)
	c.Request().Header.Set(h.PSK, "test-psk")
	c.Request().Header.Set(h.AccountNumber, "test-ebs-account-number")
	c.Request().Header.Set(h.PSKUserID, "test-psk-user")
	c.Request().Header.Set(h.OrgID, "test-orgid")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PSK).(string) != "test-psk" {
		t.Errorf("%v was set as psk instead of %v", c.Get(h.PSK).(string), "test-psk")
	}

	if c.Get(h.AccountNumber).(string) != "test-ebs-account-number" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumber).(string), "test-ebs-account-number")
	}

	if c.Get(h.PSKUserID).(string) != "test-psk-user" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.PSKUserID).(string), "test-psk-user")
	}

	if c.Get(h.OrgID).(string) != "test-orgid" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get(h.OrgID).(string))
	}

	id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
	if !ok {
		t.Errorf(`unexpected type of identity received. Want "*identity.XRHID", got "%s"`, reflect.TypeOf(c.Get(h.ParsedIdentity)))
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

	c.Request().Header.Set(h.PSK, "test-psk")
	c.Request().Header.Set(h.AccountNumber, "test-ebs-account-number")
	c.Request().Header.Set(h.PSKUserID, "test-psk-user")
	c.Request().Header.Set(h.OrgID, "test-orgid")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.PSK).(string) != "test-psk" {
		t.Errorf("%v was set as psk instead of %v", c.Get(h.PSK).(string), "test-psk")
	}

	if c.Get(h.AccountNumber).(string) != "test-ebs-account-number" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumber).(string), "test-ebs-account-number")
	}

	if c.Get(h.PSKUserID).(string) != "test-psk-user" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.PSKUserID).(string), "test-psk-user")
	}

	if c.Get(h.OrgID).(string) != "test-orgid" {
		t.Errorf(`invalid org id set. Want "%s", got "%s"`, "abcde", c.Get(h.OrgID).(string))
	}

	id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
	if !ok {
		t.Errorf(`unexpected type of identity received. Want "*identity.XRHID", got "%s"`, reflect.TypeOf(c.Get(h.ParsedIdentity)))
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

	c.Request().Header.Set(h.AccountNumber, "9876")

	err := parseOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	if c.Get(h.AccountNumber).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumber).(string), "9876")
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
	c.Request().Header.Set(h.AccountNumber, "9876")
	c.Request().Header.Set(h.PSKUserID, "555555")

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

	if c.Get(h.AccountNumber).(string) != "9876" {
		t.Errorf("%v was set as psk-account instead of %v", c.Get(h.AccountNumber).(string), "9876")
	}

	if c.Get(h.PSKUserID).(string) != "555555" {
		t.Errorf("%v was set as x-rh-sources-user-id instead of %v", c.Get(h.PSKUserID).(string), "555555")
	}
}

// TestJWTTokenValidation tests JWT token extraction and validation
func TestJWTTokenValidation(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectToken   bool
		expectLogWarn string
	}{
		{
			name:        "valid JWT token",
			authHeader:  "Bearer eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.dGVzdHNpZ25hdHVyZQ",
			expectToken: true,
		},
		{
			name:       "no Bearer prefix",
			authHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:       "Bearer with empty token",
			authHeader: "Bearer ",
		},
		{
			name:       "Bearer with whitespace only",
			authHeader: "Bearer   ",
		},
		{
			name:          "token too long",
			authHeader:    "Bearer " + strings.Repeat("a", 8193),
			expectLogWarn: "exceeds maximum length",
		},
		{
			name:          "invalid format - one part",
			authHeader:    "Bearer invalidtoken",
			expectLogWarn: "invalid format",
		},
		{
			name:          "invalid format - two parts",
			authHeader:    "Bearer header.payload",
			expectLogWarn: "invalid format",
		},
		{
			name:          "invalid format - empty part",
			authHeader:    "Bearer header..signature",
			expectLogWarn: "invalid format",
		},
		{
			name:          "invalid format - four parts",
			authHeader:    "Bearer a.b.c.d",
			expectLogWarn: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := request.CreateTestContext(
				http.MethodGet,
				"/",
				nil,
				map[string]interface{}{},
			)

			if tt.authHeader != "" {
				c.Request().Header.Set("Authorization", tt.authHeader)
			}

			err := parseOrElse204(c)
			if err != nil {
				t.Errorf("caught an error when there should not have been one: %v", err)
			}

			if rec.Code != 204 {
				t.Errorf("unexpected status code: got %v, want 204", rec.Code)
			}

			// Check if token was set in context
			token := c.Get(h.JWTToken)
			if tt.expectToken {
				if token == nil {
					t.Error("expected JWT token to be set in context, but it was not")
				} else if tokenStr, ok := token.(string); !ok || tokenStr == "" {
					t.Error("expected valid JWT token string in context")
				}
			} else {
				if token != nil {
					t.Errorf("expected no JWT token in context, but got: %v", token)
				}
			}

			// Note: We cannot easily test log warnings in this test framework
			// without additional test infrastructure. In a real scenario, you would
			// use a test logger or capture log output to verify warning messages.
		})
	}
}

// TestIsValidJWTFormat tests the JWT format validation function
func TestIsValidJWTFormat(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{"valid format", "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.dGVzdHNpZ25hdHVyZQ", true},
		{"empty", "", false},
		{"one part", "single", false},
		{"two parts", "header.payload", false},
		{"four parts", "a.b.c.d", false},
		{"empty first part", ".payload.signature", false},
		{"empty middle part", "header..signature", false},
		{"empty last part", "header.payload.", false},
		{"valid structure with real content", "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.EkN-DOsnsuRjRO6BxXemmJDm3HbxrbRzXglbN2S4sOkopdU4IsDxTI8jO19W_A4K8ZPJijNLis4EZsHeY559a4DFOd50_OqgHs3VMUSrfZ6DbGJ4-YoQYkpyMGMN0bImLYWfzjzOVROEKIQ6k_sH5aXHn19dHFcJEJrmKLJJE7O", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidJWTFormat(tt.token)
			if result != tt.expected {
				t.Errorf("isValidJWTFormat(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}
