package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// mockRbacClient helps us mock RBAC responses for testing
type mockRbacClient struct {
	allowedResponse bool
	errorResponse   error
}

func (m *mockRbacClient) Allowed(string) (bool, error) {
	if m.errorResponse != nil {
		return false, m.errorResponse
	}
	return m.allowedResponse, nil
}

// TestInternalPermissionCheck_BypassRbac tests that when bypassRbac is true, all requests are allowed
func TestInternalPermissionCheck_BypassRbac(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := InternalPermissionCheck(true, nil)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	if err != nil {
		t.Errorf("Expected no error with bypassRbac=true, got: %v", err)
	}
}

// TestInternalPermissionCheck_NoPSK tests that PSK authentication is NOT supported
func TestInternalPermissionCheck_NoPSK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set PSK in context (should be ignored by InternalPermissionCheck)
	c.Set(h.PSK, "test-psk")

	mockRbacClient := &mockRbacClient{}
	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)

	// Should fail because PSK is not supported, and no x-rh-identity provided
	if err == nil {
		t.Error("Expected error when using PSK with InternalPermissionCheck, got nil")
	}

	// Check that it's an unauthorized error
	if httpErr, ok := err.(*echo.HTTPError); ok {
		if httpErr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got: %d", httpErr.Code)
		}
	}
}

// TestInternalPermissionCheck_WithCertificate tests certificate-based authentication
func TestInternalPermissionCheck_WithCertificate(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create identity with certificate/system authentication
	xrhid := identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: "123456",
			OrgID:         "654321",
			Type:          "System",
			System: &identity.System{
				CommonName: "test-cn",
				ClusterId:  "test-cluster-id",
			},
		},
	}

	// Set parsed identity in context
	c.Set(h.ParsedIdentity, &xrhid)
	c.Set(h.XRHID, "encoded-identity")

	mockRbacClient := &mockRbacClient{}
	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	if err != nil {
		t.Errorf("Expected no error with valid certificate authentication, got: %v", err)
	}
}

// TestInternalPermissionCheck_WithUserIdentity tests regular user-based authentication with RBAC
func TestInternalPermissionCheck_WithUserIdentity(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create identity without system/certificate
	xrhid := identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: "123456",
			OrgID:         "654321",
			Type:          "User",
		},
	}

	// Encode identity
	identityJSON, _ := json.Marshal(xrhid)
	encodedIdentity := base64.StdEncoding.EncodeToString(identityJSON)

	c.Set(h.ParsedIdentity, &xrhid)
	c.Set(h.XRHID, encodedIdentity)

	// Mock RBAC client that allows the request
	mockRbacClient := &mockRbacClient{
		allowedResponse: true,
		errorResponse:   nil,
	}

	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	if err != nil {
		t.Errorf("Expected no error with RBAC-allowed user, got: %v", err)
	}
}

// TestInternalPermissionCheck_NoAuth tests that requests without any authentication are rejected
func TestInternalPermissionCheck_NoAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockRbacClient := &mockRbacClient{}
	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)

	// Should fail because no authentication provided
	if err == nil {
		t.Error("Expected error when no authentication provided, got nil")
	}

	// Verify it's the correct error
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatal("Expected echo.HTTPError")
	}

	if httpErr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got: %d", httpErr.Code)
	}

	errorDoc, ok := httpErr.Message.(util.ErrorDocument)
	if !ok {
		t.Fatal("Expected util.ErrorDocument")
	}

	expectedMessage := "Authentication required by [x-rh-identity] header"
	if errorDoc.Errors[0].Detail != expectedMessage {
		t.Errorf("Expected error message '%s', got: '%s'", expectedMessage, errorDoc.Errors[0].Detail)
	}
}
