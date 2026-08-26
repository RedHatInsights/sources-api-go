package securitylog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	l "github.com/RedHatInsights/sources-api-go/logger"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/sirupsen/logrus"
)

func setupTestLogger() *bytes.Buffer {
	buf := &bytes.Buffer{}
	l.Log = &logrus.Logger{
		Out:       buf,
		Level:     logrus.DebugLevel,
		Formatter: &logrus.JSONFormatter{},
	}
	return buf
}

func createTestContext(method string, xrhid *identity.XRHID) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(method, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if xrhid != nil {
		c.Set(h.ParsedIdentity, xrhid)
	}
	return c
}

func parseLogEntry(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse log entry: %v\nraw: %s", err, buf.String())
	}
	return result
}

func TestGetPrincipalUser(t *testing.T) {
	c := createTestContext(http.MethodGet, &identity.XRHID{
		Identity: identity.Identity{
			OrgID: "org-123",
			User:  &identity.User{UserID: "user-456"},
		},
	})

	orgID, userID, principalType := GetPrincipal(c)

	if orgID != "org-123" {
		t.Errorf("expected org_id org-123, got %s", orgID)
	}
	if userID != "user-456" {
		t.Errorf("expected user_id user-456, got %s", userID)
	}
	if principalType != "user" {
		t.Errorf("expected type user, got %s", principalType)
	}
}

func TestGetPrincipalServiceAccount(t *testing.T) {
	c := createTestContext(http.MethodGet, &identity.XRHID{
		Identity: identity.Identity{
			OrgID:          "org-789",
			ServiceAccount: &identity.ServiceAccount{ClientId: "sa-client-1"},
		},
	})

	orgID, userID, principalType := GetPrincipal(c)

	if orgID != "org-789" {
		t.Errorf("expected org_id org-789, got %s", orgID)
	}
	if userID != "sa-client-1" {
		t.Errorf("expected user_id sa-client-1, got %s", userID)
	}
	if principalType != "service_account" {
		t.Errorf("expected type service_account, got %s", principalType)
	}
}

func TestGetPrincipalSystem(t *testing.T) {
	c := createTestContext(http.MethodGet, &identity.XRHID{
		Identity: identity.Identity{
			OrgID:  "org-sys",
			System: &identity.System{CommonName: "satellite-cn"},
		},
	})

	orgID, userID, principalType := GetPrincipal(c)

	if orgID != "org-sys" {
		t.Errorf("expected org_id org-sys, got %s", orgID)
	}
	if userID != "satellite-cn" {
		t.Errorf("expected user_id satellite-cn, got %s", userID)
	}
	if principalType != "system" {
		t.Errorf("expected type system, got %s", principalType)
	}
}

func TestGetPrincipalAnonymous(t *testing.T) {
	c := createTestContext(http.MethodGet, nil)

	_, _, principalType := GetPrincipal(c)

	if principalType != "anonymous" {
		t.Errorf("expected type anonymous, got %s", principalType)
	}
}

func TestLogCrud(t *testing.T) {
	buf := setupTestLogger()

	c := createTestContext(http.MethodPost, &identity.XRHID{
		Identity: identity.Identity{
			OrgID: "12345",
			User:  &identity.User{UserID: "user-1"},
		},
	})

	LogCrud(c, "CREATE", "source", "42", "success")

	entry := parseLogEntry(t, buf)

	if entry["security_event"] != true {
		t.Error("expected security_event to be true")
	}
	if entry["action"] != "CREATE" {
		t.Errorf("expected action CREATE, got %v", entry["action"])
	}
	if entry["resource_type"] != "source" {
		t.Errorf("expected resource_type source, got %v", entry["resource_type"])
	}
	if entry["resource_id"] != "42" {
		t.Errorf("expected resource_id 42, got %v", entry["resource_id"])
	}
	if entry["outcome"] != "success" {
		t.Errorf("expected outcome success, got %v", entry["outcome"])
	}

	principal, ok := entry["principal"].(map[string]interface{})
	if !ok {
		t.Fatal("expected principal to be a map")
	}
	if principal["org_id"] != "12345" {
		t.Errorf("expected principal.org_id 12345, got %v", principal["org_id"])
	}
	if principal["user_id"] != "user-1" {
		t.Errorf("expected principal.user_id user-1, got %v", principal["user_id"])
	}
	if principal["type"] != "user" {
		t.Errorf("expected principal.type user, got %v", principal["type"])
	}
}

func TestLogStartup(t *testing.T) {
	buf := setupTestLogger()

	LogStartup("success")

	entry := parseLogEntry(t, buf)

	if entry["action"] != "STARTUP" {
		t.Errorf("expected action STARTUP, got %v", entry["action"])
	}
	if entry["resource_type"] != "process" {
		t.Errorf("expected resource_type process, got %v", entry["resource_type"])
	}
	if entry["outcome"] != "success" {
		t.Errorf("expected outcome success, got %v", entry["outcome"])
	}
}

func TestLogShutdownSuccess(t *testing.T) {
	buf := setupTestLogger()

	LogShutdown("success", "")

	entry := parseLogEntry(t, buf)

	if entry["action"] != "SHUTDOWN" {
		t.Errorf("expected action SHUTDOWN, got %v", entry["action"])
	}
	if entry["outcome"] != "success" {
		t.Errorf("expected outcome success, got %v", entry["outcome"])
	}
	if _, exists := entry["reason"]; exists {
		t.Error("expected no reason field for successful shutdown")
	}
	if entry["level"] != "info" {
		t.Errorf("expected level info for success, got %v", entry["level"])
	}
}

func TestLogShutdownFailure(t *testing.T) {
	buf := setupTestLogger()

	LogShutdown("failure", "unexpected error")

	entry := parseLogEntry(t, buf)

	if entry["outcome"] != "failure" {
		t.Errorf("expected outcome failure, got %v", entry["outcome"])
	}
	if entry["reason"] != "unexpected error" {
		t.Errorf("expected reason 'unexpected error', got %v", entry["reason"])
	}
	if entry["level"] != "error" {
		t.Errorf("expected level error for failure, got %v", entry["level"])
	}
}

func TestLogAuthFailure(t *testing.T) {
	buf := setupTestLogger()

	LogAuthFailure("invalid identity header", "192.168.1.1:12345")

	entry := parseLogEntry(t, buf)

	if entry["action"] != "AUTH_FAILURE" {
		t.Errorf("expected action AUTH_FAILURE, got %v", entry["action"])
	}
	if entry["reason"] != "invalid identity header" {
		t.Errorf("expected reason, got %v", entry["reason"])
	}
	if entry["remote_addr"] != "192.168.1.1:12345" {
		t.Errorf("expected remote_addr, got %v", entry["remote_addr"])
	}
	if entry["level"] != "warning" {
		t.Errorf("expected level warning, got %v", entry["level"])
	}
}

func TestLogAuthzFailure(t *testing.T) {
	buf := setupTestLogger()

	c := createTestContext(http.MethodPost, &identity.XRHID{
		Identity: identity.Identity{
			OrgID: "org-denied",
			User:  &identity.User{UserID: "user-denied"},
		},
	})

	LogAuthzFailure(c, "Missing RBAC permissions")

	entry := parseLogEntry(t, buf)

	if entry["action"] != "AUTHZ_FAILURE" {
		t.Errorf("expected action AUTHZ_FAILURE, got %v", entry["action"])
	}
	if entry["reason"] != "Missing RBAC permissions" {
		t.Errorf("expected reason, got %v", entry["reason"])
	}

	principal, ok := entry["principal"].(map[string]interface{})
	if !ok {
		t.Fatal("expected principal to be a map")
	}
	if principal["org_id"] != "org-denied" {
		t.Errorf("expected principal.org_id org-denied, got %v", principal["org_id"])
	}
}

func TestIsMutatingMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodPatch, true},
		{http.MethodDelete, true},
		{http.MethodGet, false},
		{http.MethodHead, false},
		{http.MethodOptions, false},
	}

	for _, tt := range tests {
		if got := IsMutatingMethod(tt.method); got != tt.expected {
			t.Errorf("IsMutatingMethod(%s) = %v, want %v", tt.method, got, tt.expected)
		}
	}
}

func TestFormatID(t *testing.T) {
	if got := FormatID(42); got != "42" {
		t.Errorf("FormatID(42) = %s, want 42", got)
	}
	if got := FormatID(0); got != "0" {
		t.Errorf("FormatID(0) = %s, want 0", got)
	}
}
