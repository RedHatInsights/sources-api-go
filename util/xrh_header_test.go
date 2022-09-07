package util

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/kafka"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// ACCOUNT_NUMBER to be used in the tests.
const ACCOUNT_NUMBER = "12345"

// ORG_ID to be used in the tests.
const ORG_ID = "abc-org-id"

// setUpValidIdentity returns a base64 encoded valid identity.
func setUpValidIdentity(t *testing.T) string {
	xRhId := identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: ACCOUNT_NUMBER,
			OrgID:         ORG_ID,
		},
	}

	jsonIdentity, err := json.Marshal(xRhId)
	if err != nil {
		t.Errorf(`could not marshal test identity to JSON: %s`, err)
	}

	base64Identity := base64.StdEncoding.EncodeToString(jsonIdentity)

	return base64Identity
}

// TestParseXRHIDHeader tests that the function parses the xRhIdentity header correctly.
func TestParseXRHIDHeader(t *testing.T) {
	base64Identity := setUpValidIdentity(t)

	// Call the function under test.
	result, err := ParseXRHIDHeader(base64Identity)
	if err != nil {
		t.Errorf(`unexpected error when parsing the identity: %s`, err)
	}

	{
		want := ORG_ID
		got := result.Identity.OrgID

		if want != got {
			t.Errorf(`invalid OrgId returned. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := ACCOUNT_NUMBER
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

// TestIdentityFromKafkaHeaders tests that the function under test is able to extract the identity when the
// "x-rh-identity" is given, and when the "x-rh-account-number" header is given.
func TestIdentityFromKafkaHeaders(t *testing.T) {
	// First test with the "x-rh-identity" header.
	base64Identity := setUpValidIdentity(t)

	headers := []kafka.Header{
		{
			Key:   xrhIdentityKey,
			Value: []byte(base64Identity),
		},
	}

	// Call the function under test.
	id, err := IdentityFromKafkaHeaders(headers)
	if err != nil {
		t.Errorf(`unexpected error when recovering the identity: %s`, err)
	}

	{
		want := ACCOUNT_NUMBER
		got := id.AccountNumber
		if want != got {
			t.Errorf(`invalid account number extracted from identity. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := ORG_ID
		got := id.OrgID
		if want != got {
			t.Errorf(`invalid orgId extracted from identity. Want "%s", got "%s"`, want, got)
		}
	}

	// Lastly test with the "x-rh-account-number" and "x-rh-sources-org-id" header.
	headers = []kafka.Header{
		{
			Key:   xrhAccountNumberKey,
			Value: []byte(ACCOUNT_NUMBER),
		},
		{
			Key:   h.ORGID,
			Value: []byte(ORG_ID),
		},
	}

	id, err = IdentityFromKafkaHeaders(headers)
	if err != nil {
		t.Errorf(`unexpected error when recovering the identity: %s`, err)
	}

	{
		want := ACCOUNT_NUMBER
		got := id.AccountNumber
		if want != got {
			t.Errorf(`invalid account number extracted from identity. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := ORG_ID
		got := id.OrgID
		if want != got {
			t.Errorf(`invalid org id extracted from identity. Want "%s", got "%s"`, want, got)
		}
	}

}
