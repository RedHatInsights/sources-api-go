package util

import (
	"encoding/base64"
	"encoding/json"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"strings"
	"testing"
)

// TestParseXRHIDHeader tests that the function parses the xRhIdentity header correctly.
func TestParseXRHIDHeader(t *testing.T) {
	// Set up the identity with some custom information.
	accountNumber := "12345"
	orgId := "abc-org-id"

	xRhId := identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: accountNumber,
			OrgID:         orgId,
		},
	}

	jsonIdentity, err := json.Marshal(xRhId)
	if err != nil {
		t.Errorf(`could not marshal test identity to JSON: %s`, err)
	}

	base64Identity := base64.StdEncoding.EncodeToString(jsonIdentity)

	// Call the function under test.
	result, err := ParseXRHIDHeader(base64Identity)
	if err != nil {
		t.Errorf(`unexpected error when parsing the identity: %s`, err)
	}

	{
		want := orgId
		got := result.Identity.OrgID

		if want != got {
			t.Errorf(`invalid OrgId returned. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := accountNumber
		got := result.Identity.AccountNumber

		if want != got {
			t.Errorf(`invalid account number returned. Want "%s", got "%s"`, want, got)
		}
	}

}

// TestParseXRHIDHeaderInvalidBase64String tests that an error is returned when the given string is not properly base64
// encoded.
func TestParseXRHIDHeaderInvalidBase64String(t *testing.T) {
	invalidIdentity := "Hello, World!"

	_, got := ParseXRHIDHeader(invalidIdentity)

	want := "error decoding Identity"
	if !strings.Contains(got.Error(), want) {
		t.Errorf(`unexpected error received when decoding an invalid base64 string. Want "%s", got "%s"`, want, got)
	}
}

// TestParseXRHIDHeaderInvalidBase64Json tests that an error is returned when the given string has an invalid base64
// encoded JSON.
func TestParseXRHIDHeaderInvalidBase64Json(t *testing.T) {
	// {"hello": "world"
	invalidJson := "eyJoZWxsbyI6ICJ3b3JsZCI="

	_, got := ParseXRHIDHeader(invalidJson)

	want := "x-rh-identity header does not contain"
	if !strings.Contains(got.Error(), want) {
		t.Errorf(`unexpected error received when decoding an invalid base64 string. Want "%s", got "%s"`, want, got)
	}
}
