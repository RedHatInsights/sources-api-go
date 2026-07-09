package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// Note: mockRbacClient is defined in authorization_test.go and shared across all middleware tests

// TestInternalPermissionCheck_BypassRbac tests that when bypassRbac is true, all requests are allowed
func TestInternalPermissionCheck_BypassRbac(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/sources/v2/secrets/123", nil)
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set PSK in context (should be ignored by InternalPermissionCheck)
	c.Set(h.PSK, "test-psk")

	mockRbacClient := &mockRbacClient{mockedRbacResponse: mockedRbacResponse{}}
	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)

	// Should not return an error (c.JSON returns nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that it returned 401 Unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got: %d", rec.Code)
	}
}

// TestInternalPermissionCheck_WithCertificate tests certificate-based authentication
func TestInternalPermissionCheck_WithCertificate(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/sources/v2/secrets/123", nil)
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

	mockRbacClient := &mockRbacClient{mockedRbacResponse: mockedRbacResponse{}}
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/sources/v2/secrets/123", nil)
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
		mockedRbacResponse: mockedRbacResponse{
			AllowedResponse: true,
			ErrorResponse:   nil,
		},
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/sources/v2/secrets/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockRbacClient := &mockRbacClient{mockedRbacResponse: mockedRbacResponse{}}
	middleware := InternalPermissionCheck(false, mockRbacClient)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)

	// Should not return an error (c.JSON returns nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify it returned 401 Unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got: %d", rec.Code)
	}
}
