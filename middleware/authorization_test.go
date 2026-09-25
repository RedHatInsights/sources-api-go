package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// mockedRbacResponse defines the response that we will get from the RBAC client.
type mockedRbacResponse struct {
	AllowedResponse bool
	ErrorResponse   error
}

// mockRbacClient helps us mock RBAC responses.
type mockRbacClient struct {
	mockedRbacResponse
}

func (m *mockRbacClient) Allowed(string) (bool, error) {
	if m.ErrorResponse != nil {
		return false, m.ErrorResponse
	}

	return m.AllowedResponse, nil
}

// setUpMiddleware sets up a "PermissionCheck" middleware with the given arguments. It also sets the middleware up so
// that if no errors occur, a "204 — No content" response is returned.
func setUpMiddleware(bypassRbac bool, allowedPsks []string, rbacResponse mockedRbacResponse) echo.HandlerFunc {
	mockedRbacClient := mockRbacClient{mockedRbacResponse: rbacResponse}

	middleware := PermissionCheck(bypassRbac, allowedPsks, &mockedRbacClient)

	return middleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
}

func TestRbacDisabled(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{},
	)

	middleware := setUpMiddleware(true, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestPSKMatches(t *testing.T) {
	allowedPsks := []string{"1234"}

	if pskMatches(allowedPsks, "1234") != true {
		t.Errorf("psk didn't match when it should have")
	}

	if pskMatches(allowedPsks, "12345") == true {
		t.Errorf("psk matched when it should not have")
	}
}

func TestGoodPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	middleware := setUpMiddleware(true, []string{"1234"}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestBadPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	middleware := setUpMiddleware(false, []string{"abcdef"}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestNoPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{},
	)

	middleware := setUpMiddleware(false, []string{"abcdef"}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestSystemClusterID(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						ClusterId: "test_cluster",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(true, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestSystemCN(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(true, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

// TestSystemMethodNotAllowed tests that "405 — Method not allowed" errors are
// returned when using disallowed methods with System or certificate based
// authentications.
func TestSystemMethodNotAllowed(t *testing.T) {
	testMethods := []string{
		http.MethodConnect,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPut,
		http.MethodTrace,
	}

	for _, method := range testMethods {
		c, rec := request.CreateTestContext(
			method,
			"/",
			nil,
			map[string]interface{}{
				h.XRHID: "dummy",
				h.ParsedIdentity: &identity.XRHID{
					Identity: identity.Identity{
						System: &identity.System{
							CommonName: "test_cert",
						},
					},
				},
			},
		)

		middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{})

		err := middleware(c)
		if err != nil {
			t.Errorf(`want no error, got "%s"`, err)
		}

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf(`want status code "%d", got "%d"`, http.StatusMethodNotAllowed, rec.Code)
		}
	}
}

func TestSystemDelete(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSystemDeleteSource(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/sources/1235",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestSystemDeleteSourceVersioned(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/1235",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestRbacWithAccess(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{AllowedResponse: true})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestRbacWithoutAccess(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{AllowedResponse: false})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestRbacNoConnection(t *testing.T) {
	c, _ := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{ErrorResponse: fmt.Errorf("unable to connect to rbac")})

	err := middleware(c)
	if err == nil {
		t.Errorf("no error was returned when we were expecting one!")
	}
}

// TestIsUsingCertificateBasedAuthentication tests that the function under test
// correctly identifies a certificate based authentication.
func TestIsUsingCertificateBasedAuthentication(t *testing.T) {
	// An empty identity simulates that no "system" object was present in the
	// JSON object.
	if isUsingCertificateBasedAuthentication(&identity.XRHID{Identity: identity.Identity{}}) {
		t.Error(`the function under test incorrectly identified a "nil" "System" struct as a certificate based authentication`)
	}

	// An empty "System" struct simulates that the "system" JSON object was
	// defined but empty, like so: "system": {}
	if isUsingCertificateBasedAuthentication(&identity.XRHID{Identity: identity.Identity{System: &identity.System{}}}) {
		t.Error(`the function under test incorrectly identified a default "System" struct as a certificate based authentication`)
	}

	// Tests that non-empty "System" structs are correctly identified as
	// certificate based authentications.
	certificateBasedAuthenticationStructs := []*identity.XRHID{
		{Identity: identity.Identity{System: &identity.System{CertType: "ct"}}},
		{Identity: identity.Identity{System: &identity.System{ClusterId: "cid"}}},
		{Identity: identity.Identity{System: &identity.System{CommonName: "cn"}}},
	}

	for _, cbas := range certificateBasedAuthenticationStructs {
		if !isUsingCertificateBasedAuthentication(cbas) {
			t.Errorf(`the given identity struct was not correctly identified as a system based authentication: %#v`, cbas)
		}
	}
}

// TestInvalidPSKType tests that when the PSK in the context is not a string,
// a generic error message is returned.
func TestInvalidPSKType(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]any{},
	)

	// Set PSK as an invalid type (int instead of string)
	c.Set(h.PSK, 12345)

	middleware := setUpMiddleware(false, []string{"valid-psk"}, mockedRbacResponse{
		AllowedResponse: false,
		ErrorResponse:   nil,
	})

	err := middleware(c)

	// Should return an error with generic message
	if err == nil {
		t.Error("expected an error but got nil")
	}

	// The error message should be generic, not expose type details
	errMsg := err.Error()
	if strings.Contains(errMsg, "casting") || strings.Contains(errMsg, "type") || strings.Contains(errMsg, "int") {
		t.Errorf("error message should be generic, but got: %s", errMsg)
	}

	// Check that it contains the generic message
	if !strings.Contains(errMsg, "authentication failed") {
		t.Errorf("expected generic 'authentication failed' message, got: %s", errMsg)
	}
}

// TestInvalidParsedIdentityType tests that when the parsed identity in the context
// is not an *identity.XRHID, a generic error message is returned.
func TestInvalidParsedIdentityType(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]any{},
	)

	// Set x-rh-identity header and parsed identity with wrong type
	c.Set(h.XRHID, "some-header-value")
	c.Set(h.ParsedIdentity, "invalid-identity-type")

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{
		AllowedResponse: false,
		ErrorResponse:   nil,
	})

	err := middleware(c)

	// Should return an error with generic message
	if err == nil {
		t.Error("expected an error but got nil")
	}

	// The error message should be generic, not expose type details
	errMsg := err.Error()
	if strings.Contains(errMsg, "casting") || strings.Contains(errMsg, "struct") || strings.Contains(errMsg, "XRHID") || strings.Contains(errMsg, "string") {
		t.Errorf("error message should be generic, but got: %s", errMsg)
	}

	// Check that it contains the generic message
	if !strings.Contains(errMsg, "authentication failed") {
		t.Errorf("expected generic 'authentication failed' message, got: %s", errMsg)
	}
}

// TestInvalidXRHIDHeaderType tests that when the x-rh-identity header in the context
// is not a string, a generic error message is returned.
func TestInvalidXRHIDHeaderType(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]any{},
	)

	// Set x-rh-identity header with wrong type (int instead of string)
	c.Set(h.XRHID, 12345)
	c.Set(h.ParsedIdentity, &identity.XRHID{
		Identity: identity.Identity{
			AccountNumber:         "123",
			OrgID:                 "456",
			EmployeeAccountNumber: "",
			Internal: identity.Internal{
				OrgID:       "",
				AuthTime:    0,
				CrossAccess: false,
			},
			User:           nil,
			System:         nil,
			Associate:      nil,
			X509:           nil,
			ServiceAccount: nil,
			Type:           "",
			AuthType:       "",
		},
		Entitlements: nil,
	})

	middleware := setUpMiddleware(false, []string{}, mockedRbacResponse{
		AllowedResponse: true,
		ErrorResponse:   nil,
	})

	err := middleware(c)

	// Should return an error with generic message
	if err == nil {
		t.Error("expected an error but got nil")
	}

	// The error message should be generic, not expose type details
	errMsg := err.Error()
	if strings.Contains(errMsg, "string") || strings.Contains(errMsg, "casting") || strings.Contains(errMsg, "header") || strings.Contains(errMsg, "int") {
		t.Errorf("error message should be generic, but got: %s", errMsg)
	}

	// Check that it contains the generic message
	if !strings.Contains(errMsg, "authentication failed") {
		t.Errorf("expected generic 'authentication failed' message, got: %s", errMsg)
	}
}
