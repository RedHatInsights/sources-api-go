package util

import (
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v4"
)

// (compile-time check to make sure our context behaves like an echo.Context)
var _ = (echo.Context)(&SourcesContext{})

// TestGetTenantFromEchoContext tests that the tenant id is correctly pulled from the context when the tenant has been
// passed correctly.
func TestGetTenantFromEchoContext(t *testing.T) {
	want := int64(12345)

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []Filter{},
			"tenantID": want,
		},
	)

	got, err := GetTenantFromEchoContext(c)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if want != got {
		t.Errorf(`incorrect tenant pulled. Want "%d", got "%d"`, want, got)
	}
}

// TestGetTenantFromEchoContextBelowEqualsZero tests that when passed a tenant ID which is zero or lower than that,
// a proper error is returned.
func TestGetTenantFromEchoContextLowerOrEqualsZero(t *testing.T) {
	invalidTenantIds := []int64{-5, 0}

	for _, iti := range invalidTenantIds {
		c, _ := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/whatever",
			nil,
			map[string]interface{}{
				"limit":    100,
				"offset":   0,
				"filters":  []Filter{},
				"tenantID": iti,
			},
		)

		_, err := GetTenantFromEchoContext(c)

		want := "incorrect tenant value provided"
		if !strings.Contains(want, err.Error()) {
			t.Errorf(`want "%s", got "%s"`, want, err)
		}
	}
}

// TestGetTenantFromEchoContextInvalidFormat tests that when a tenant is given in an invalid format, an error is
// returned.
func TestGetTenantFromEchoContextInvalidFormat(t *testing.T) {
	invalidTenantIdFormat := "12345"

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []Filter{},
			"tenantID": invalidTenantIdFormat,
		},
	)

	want := "the tenant was provided in an invalid format"
	_, err := GetTenantFromEchoContext(c)
	if !strings.Contains(want, err.Error()) {
		t.Errorf(`incorrect tenant pulled. Want "%s", got "%s"`, want, err)
	}
}

// TestGetTenantFromEchoContextMissing tests that when the tenant is missing from the context the function returns a
// default value and a nil error.
func TestGetTenantFromEchoContextMissing(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []Filter{},
		},
	)

	want := int64(0)
	got, err := GetTenantFromEchoContext(c)

	if want != got {
		t.Errorf(`incorrect tenant pulled. Want "%d", got "%d`, want, got)
	}

	if err != nil {
		t.Errorf(`want nil err, got "%s"`, err)
	}
}
