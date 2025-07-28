package middleware

import (
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTAuthentication_FeatureFlagDisabled(t *testing.T) {
	service.ClearTestFeatureFlags()

	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, false)
	defer service.ClearTestFeatureFlags()

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			h.JWTToken: "some-jwt-token",
		},
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
	assert.Nil(t, c.Get(h.JWTSubject))
}

func TestJWTAuthentication_NoToken(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{}, // No JWT token
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
	assert.Nil(t, c.Get(h.JWTSubject))
}

func TestJWTAuthentication_EmptyToken(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			h.JWTToken: "", // Empty token
		},
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
	assert.Nil(t, c.Get(h.JWTSubject))
}

func TestJWTAuthentication_InvalidToken(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			h.JWTToken: "invalid.jwt.token",
		},
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication failed")
	assert.Nil(t, c.Get(h.JWTSubject))
}

func TestValidateJWTSubject(t *testing.T) {
	tests := []struct {
		name        string
		subject     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid subject",
			subject:     "user@example.com",
			expectError: false,
		},
		{
			name:        "empty subject",
			subject:     "",
			expectError: true,
			errorMsg:    "missing subject",
		},
		{
			name:        "subject too long",
			subject:     strings.Repeat("a", 300),
			expectError: true,
			errorMsg:    "subject too long: 300 bytes (max 256)",
		},
		{
			name:        "subject at limit",
			subject:     strings.Repeat("a", 256),
			expectError: false,
		},
		{
			name:        "subject just over limit",
			subject:     strings.Repeat("a", 257),
			expectError: true,
			errorMsg:    "subject too long: 257 bytes (max 256)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSubject(tt.subject)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
