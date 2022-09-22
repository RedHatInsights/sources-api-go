package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/kafka"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// TestForwadableHeadersPskAccountNumber tests that when the "psk-account" context value is present, two headers are returned from
// the function under test: "x-rh-sources-account-number" and "x-rh-identity".
func TestForwadableHeadersAccountNumber(t *testing.T) {
	testPskAccountValue := "abcde"

	context, _ := request.CreateTestContext("GET", "https://example.org/hello", nil, nil)
	context.Set(h.AccountNumber, testPskAccountValue)

	// Call the function under test.
	headers, err := ForwadableHeaders(context)
	if err != nil {
		t.Errorf(`unexpected error when building the forwardable headers: %s`, err)
	}

	{
		want := 2
		got := len(headers)

		if want != got {
			t.Errorf(`incorrect number of Kafka headers generated. Want "%d", got "%d"`, want, got)
		}
	}

	{
		pskHeader := headers[0]

		{
			want := "x-rh-sources-account-number"
			got := pskHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			want := []byte(testPskAccountValue)
			got := pskHeader.Value

			if !bytes.Equal(want, got) {
				t.Errorf(`incorrect Kafka header value generated. Want "%s", got "%s"`, want, got)
			}
		}
	}
	{
		xRhIdHeader := headers[1]
		{
			want := h.XRHID
			got := xRhIdHeader.Key
			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			id, err := util.IdentityFromKafkaHeaders([]kafka.Header{xRhIdHeader})
			if err != nil {
				t.Errorf(`unexpected error when extracting the identity from the Kafka header: %s`, err)
			}

			want := testPskAccountValue
			got := id.AccountNumber

			if want != got {
				t.Errorf(`incorrect Kafka header value generated. Want "%s", got "%s"`, want, got)
			}
		}
	}
}

// TestForwadableHeadersOrgId tests that when the "x-rh-sources-org-id" context value is present, two headers are
// returned from the function under test: "x-rh-sources-org-id" and "x-rh-identity".
func TestForwadableHeadersOrgId(t *testing.T) {
	testOrgIdValue := "abcde"

	context, _ := request.CreateTestContext("GET", "https://example.org/hello", nil, nil)
	context.Set(h.OrgID, testOrgIdValue)

	// Call the function under test.
	headers, err := ForwadableHeaders(context)
	if err != nil {
		t.Errorf(`unexpected error when building the forwardable headers: %s`, err)
	}

	{
		want := 2
		got := len(headers)

		if want != got {
			t.Errorf(`incorrect number of Kafka headers generated. Want "%d", got "%d"`, want, got)
		}
	}

	{
		orgIdHeader := headers[0]

		{
			want := h.OrgID
			got := orgIdHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			want := []byte(testOrgIdValue)
			got := orgIdHeader.Value

			if !bytes.Equal(want, got) {
				t.Errorf(`incorrect Kafka header value generated. Want "%s", got "%s"`, want, got)
			}
		}
	}
	{
		xRhIdHeader := headers[1]
		{
			want := h.XRHID
			got := xRhIdHeader.Key
			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			id, err := util.IdentityFromKafkaHeaders([]kafka.Header{xRhIdHeader})
			if err != nil {
				t.Errorf(`unexpected error when extracting the identity from the Kafka header: %s`, err)
			}

			want := testOrgIdValue
			got := id.OrgID

			if want != got {
				t.Errorf(`incorrect Kafka header value generated. Want "%s", got "%s"`, want, got)
			}
		}
	}
}

// TestForwadableHeadersXrhId tests that when the "x-rh-identity" context value
// is present, it passes it along as well as adding the psk-account + psk-org-id
// related headers.
func TestForwadableHeadersXrhId(t *testing.T) {
	context, _ := request.CreateTestContext("GET", "https://example.org/hello", nil, nil)

	// Generate the XRHID to set it in the context.
	var xRhId identity.XRHID

	testAccountNumber := "abcde"
	testOrgId := "12345"

	xRhId.Identity.AccountNumber = testAccountNumber
	xRhId.Identity.OrgID = testOrgId

	result, err := json.Marshal(xRhId)
	if err != nil {
		t.Errorf(`unexpected error when marshalling the XRHID: %s`, err)
	}

	context.Set(h.XRHID, base64.StdEncoding.EncodeToString(result))

	// Call the function under test.
	headers, err := ForwadableHeaders(context)
	if err != nil {
		t.Errorf(`unexpected error when building the forwardable headers: %s`, err)
	}

	{
		want := 3
		got := len(headers)

		if want != got {
			t.Errorf(`incorrect number of Kafka headers generated. Want "%d", got "%d"`, want, got)
		}
	}

	{
		accountHeader := headers[0]
		{
			want := "x-rh-sources-account-number"
			got := accountHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			{
				want := testAccountNumber
				got := string(accountHeader.Value)

				if want != got {
					t.Errorf(`incorrect account number on xRhId struct. Want "%s", got "%s"`, want, got)
				}
			}
		}
	}

	{
		orgIdHeader := headers[1]
		{
			want := h.OrgID
			got := orgIdHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			want := testOrgId
			got := string(orgIdHeader.Value)

			if want != got {
				t.Errorf(`incorrect orgId on xRhId struct. Want "%s", got "%s"`, want, got)
			}
		}
	}

	{
		xRhIdentityHeader := headers[2]

		{
			want := h.XRHID
			got := xRhIdentityHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			xRhId, err := util.ParseXRHIDHeader(string(xRhIdentityHeader.Value))
			if err != nil {
				t.Errorf(`unexpected error when parsing the xRhIdentity base64 string: %s`, err)
			}
			{
				want := testAccountNumber
				got := xRhId.Identity.AccountNumber

				if want != got {
					t.Errorf(`incorrect account number on xRhId struct. Want "%s", got "%s"`, want, got)
				}
			}
			{
				want := testOrgId
				got := xRhId.Identity.OrgID

				if want != got {
					t.Errorf(`incorrect orgId on xRhId struct. Want "%s", got "%s"`, want, got)
				}
			}
		}
	}
}

// TestForwadableHeadersOrgId tests that when the and "x-rh-sources-org-id" context value is present,
// two headers are returned from the function under test: "x-rh-sources-org-id" and "x-rh-identity"
func TestForwadableHeadersPskOrgId(t *testing.T) {
	testOrgIdValue := "12345"

	context, _ := request.CreateTestContext("GET", "https://example.org/hello", nil, nil)
	context.Set(h.OrgID, testOrgIdValue)

	// Call the function under test.
	headers, err := ForwadableHeaders(context)
	if err != nil {
		t.Errorf(`unexpected error when building the forwardable headers: %s`, err)
	}

	{
		want := 2
		got := len(headers)

		if want != got {
			t.Errorf(`incorrect number of Kafka headers generated. Want "%d", got "%d"`, want, got)
		}
	}

	{
		orgIdHeader := headers[0]

		{
			want := h.OrgID
			got := orgIdHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			want := []byte(testOrgIdValue)
			got := orgIdHeader.Value

			if !bytes.Equal(want, got) {
				t.Errorf(`incorrect Kafka header value generated. Want "%s", got "%s"`, want, got)
			}
		}
	}

	{
		xRhIdentityHeader := headers[1]

		{
			want := h.XRHID
			got := xRhIdentityHeader.Key

			if want != got {
				t.Errorf(`incorrect Kafka header generated. Want "%s", got "%s"`, want, got)
			}
		}
		{
			xRhId, err := util.ParseXRHIDHeader(string(xRhIdentityHeader.Value))
			if err != nil {
				t.Errorf(`unexpected error when parsing the xRhIdentity base64 string: %s`, err)
			}
			{
				want := testOrgIdValue
				got := xRhId.Identity.OrgID

				if want != got {
					t.Errorf(`incorrect orgId on xRhId struct. Want "%s", got "%s"`, want, got)
				}
			}
		}
	}

}
