package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/middleware/oidc"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJWTRbacClient is a simple mock for JWT testing
type mockJWTRbacClient struct{}

func (m *mockJWTRbacClient) Allowed(xRhIdHeader string) (bool, error) {
	return true, nil
}

func TestJWTAuthenticationMiddleware_Success(t *testing.T) {
	// Enable OIDC feature flag for this test
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	// Generate test keys
	privateKey, publicKeyPEM, err := oidc.GenerateTestKeyPair()
	require.NoError(t, err)

	// Create test claims
	claims := jwt.MapClaims{
		"sub": "test-service-user",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	// Create test JWT
	tokenString, err := oidc.CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	// Set up test environment variables
	t.Setenv("JWT_PUBLIC_KEY", publicKeyPEM)
	t.Setenv("JWT_ISSUER", "test-issuer")
	t.Setenv("JWT_AUDIENCE", "test-audience")

	// Create Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Parse headers first (simulating the middleware chain)

	err = ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		// Verify JWT user ID is set
		userID := c.Get(h.JWTUserID)
		assert.Equal(t, "test-service-user", userID)

		return c.String(http.StatusOK, "success")
	})(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuthenticationMiddleware_InvalidToken(t *testing.T) {
	// Enable OIDC feature flag for this test
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	_, publicKeyPEM, err := oidc.GenerateTestKeyPair()
	require.NoError(t, err)

	// Set up test environment variables
	t.Setenv("JWT_PUBLIC_KEY", publicKeyPEM)
	t.Setenv("JWT_ISSUER", "test-issuer")
	t.Setenv("JWT_AUDIENCE", "test-audience")

	// Create Echo context with invalid token
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Parse headers first
	err = ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})(c)

	// Should not return error but set HTTP 401
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid JWT token")
}

func TestJWTAuthenticationMiddleware_MissingConfiguration(t *testing.T) {
	// Enable OIDC feature flag for this test
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	// Generate test keys but don't configure them
	privateKey, _, err := oidc.GenerateTestKeyPair()
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "test-service-user",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	tokenString, err := oidc.CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	// Set up test environment without JWT config (empty values)
	t.Setenv("JWT_PUBLIC_KEY", "")
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("JWT_AUDIENCE", "")

	// Create Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Parse headers first
	err = ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid JWT token")
}

func TestJWTAuthenticationMiddleware_FallsBackToOtherAuth(t *testing.T) {
	// Create Echo context without JWT but with PSK
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-rh-sources-psk", "valid-psk")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Parse headers first
	err := ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware with valid PSK
	middleware := PermissionCheck(false, []string{"valid-psk"}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", strings.TrimSpace(rec.Body.String()))
}

func TestJWTAuthenticationMiddleware_NoAuthProvided(t *testing.T) {
	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	// Create Echo context without any authentication
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Parse headers first
	err := ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication required by either")
}

func TestJWTAuthenticationMiddleware_FeatureFlagDisabled(t *testing.T) {
	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	// Generate test keys
	privateKey, publicKeyPEM, err := oidc.GenerateTestKeyPair()
	require.NoError(t, err)

	// Set up JWT configuration with the test public key
	t.Setenv("JWT_PUBLIC_KEY", publicKeyPEM)
	t.Setenv("JWT_ENABLED", "true")

	// Reload config with environment variables
	config.Reset()

	// Create test claims
	claims := jwt.MapClaims{
		"sub": "test-user-123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Create test JWT token
	tokenString, err := oidc.CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	// Create test request with JWT token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()

	// Create Echo context
	e := echo.New()
	c := e.NewContext(req, rec)

	// Parse headers first (simulating the middleware chain)
	err = ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		// If we reach here, authentication was successful
		// Since OIDC is disabled by default (feature flag), it should fall through to other auth methods
		// and since we don't have XRHID, it should require authentication
		return c.JSON(http.StatusOK, map[string]string{"status": "authenticated"})
	})(c)

	// With OIDC feature flag disabled by default, JWT token should be ignored
	// and authentication should fail since no XRHID is provided
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication required by either")
}

func TestJWTAuthenticationMiddleware_FeatureFlagBehavior(t *testing.T) {
	// Reset config to allow fresh loading with test environment variables
	config.Reset()

	// Generate test keys
	privateKey, publicKeyPEM, err := oidc.GenerateTestKeyPair()
	require.NoError(t, err)

	// Set up JWT configuration with the test public key
	t.Setenv("JWT_PUBLIC_KEY", publicKeyPEM)
	t.Setenv("JWT_ENABLED", "true")

	// Reload config with environment variables
	config.Reset()

	// Create test claims
	claims := jwt.MapClaims{
		"sub": "test-user-123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Create test JWT token
	tokenString, err := oidc.CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	// Create test request with JWT token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()

	// Create Echo context
	e := echo.New()
	c := e.NewContext(req, rec)

	// Parse headers first (simulating the middleware chain)
	err = ParseHeaders(func(c echo.Context) error {
		return nil
	})(c)
	require.NoError(t, err)

	// Verify that JWT token was parsed and stored in context
	jwtToken := c.Get(h.JWTToken)
	assert.NotNil(t, jwtToken, "JWT token should be parsed and stored in context")
	assert.Equal(t, tokenString, jwtToken, "JWT token should match the provided token")

	// Test the permission check middleware
	middleware := PermissionCheck(false, []string{}, &mockJWTRbacClient{})

	err = middleware(func(c echo.Context) error {
		// Check that no JWT user ID was set since OIDC feature is disabled
		jwtUserID := c.Get(h.JWTUserID)
		assert.Nil(t, jwtUserID, "JWT User ID should not be set when OIDC feature flag is disabled")

		return c.JSON(http.StatusOK, map[string]string{"status": "checked"})
	})(c)

	// The test should complete without error, verifying that JWT token is parsed
	// but not used for authentication when feature flag is disabled
	require.NoError(t, err)
}
