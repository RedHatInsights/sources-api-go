package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock GetJWKS function for testing
var originalGetJWKS func(context.Context, string) (jwk.Set, error)

func mockGetJWKS(keySet jwk.Set, err error) {
	originalGetJWKS = GetJWKS
	GetJWKS = func(ctx context.Context, issuer string) (jwk.Set, error) {
		return keySet, err
	}
}

func restoreGetJWKS() {
	if originalGetJWKS != nil {
		GetJWKS = originalGetJWKS
		originalGetJWKS = nil
	}
}

func createTestJWK() (jwk.Key, *rsa.PrivateKey, error) {
	// Generate a new RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create JWK from the generated key's public key
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	key.Set(jwk.KeyIDKey, "test-key-id")
	key.Set(jwk.AlgorithmKey, jwa.RS256)
	key.Set(jwk.KeyUsageKey, "sig")

	return key, privateKey, nil
}

func createSignedJWT(issuer, subject string, privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()

	token, err := jwt.NewBuilder().
		Issuer(issuer).
		Subject(subject).
		IssuedAt(now).
		Expiration(now.Add(time.Hour)).
		Build()
	if err != nil {
		return "", err
	}

	// Create a private key for signing
	signingKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		return "", err
	}

	signingKey.Set(jwk.KeyIDKey, "test-key-id")
	signingKey.Set(jwk.AlgorithmKey, jwa.RS256)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signingKey))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func TestJWTAuthentication_FeatureFlagDisabled(t *testing.T) {
	service.ClearTestFeatureFlags()

	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, false)
	defer service.ClearTestFeatureFlags()

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			"headers": map[string]string{
				"Authorization": "Bearer some-jwt-token",
			},
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
			"headers": map[string]string{
				"Authorization": "Bearer ", // Empty token
			},
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
			"headers": map[string]string{
				"Authorization": "Bearer invalid.jwt.token",
			},
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

func setupTestEnv(t *testing.T, issuer string) func() {
	originalIssuer := os.Getenv("JWT_ISSUER")

	if issuer != "" {
		os.Setenv("JWT_ISSUER", issuer)
	} else {
		os.Unsetenv("JWT_ISSUER")
	}

	config.Reset()

	return func() {
		if originalIssuer != "" {
			os.Setenv("JWT_ISSUER", originalIssuer)
		} else {
			os.Unsetenv("JWT_ISSUER")
		}

		config.Reset()
	}
}

func createMockEchoContext() echo.Context {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{},
	)

	return c
}

func TestValidateJWT_TokenParsingFailure(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	tests := []struct {
		name        string
		token       string
		expectError string
	}{
		{
			name:        "malformed token",
			token:       "not.a.jwt",
			expectError: "JWT parsing failed",
		},
		{
			name:        "empty token",
			token:       "",
			expectError: "JWT parsing failed",
		},
		{
			name:        "invalid base64",
			token:       "invalid.base64!.token",
			expectError: "JWT parsing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			c := createMockEchoContext()
			err := validateJWT(ctx, c, tt.token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestValidateJWT_IssuerValidation(t *testing.T) {
	tests := []struct {
		name         string
		configIssuer string
		tokenIssuer  string
		expectError  string
		shouldFail   bool
	}{
		{
			name:         "missing token issuer",
			configIssuer: "https://example.com",
			tokenIssuer:  "",
			expectError:  "invalid JWT issuer: missing issuer",
			shouldFail:   true,
		},
		{
			name:         "issuer mismatch",
			configIssuer: "https://example.com",
			tokenIssuer:  "https://wrong.com",
			expectError:  "invalid JWT issuer: expected https://example.com, got https://wrong.com",
			shouldFail:   true,
		},
		{
			name:         "matching issuer",
			configIssuer: "https://example.com",
			tokenIssuer:  "https://example.com",
			shouldFail:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t, tt.configIssuer)
			defer cleanup()

			// Create a basic JWT with the specified issuer
			header := map[string]interface{}{
				"alg": "HS256",
				"typ": "JWT",
			}
			payload := map[string]interface{}{
				"iss": tt.tokenIssuer,
				"sub": "test-subject",
				"exp": time.Now().Add(time.Hour).Unix(),
				"iat": time.Now().Unix(),
			}

			headerBytes, _ := json.Marshal(header)
			payloadBytes, _ := json.Marshal(payload)

			headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
			payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

			// Create a token with dummy signature (will fail at JWKS step if validation passes)
			token := headerB64 + "." + payloadB64 + ".dummy-signature"

			ctx := context.Background()
			c := createMockEchoContext()
			err := validateJWT(ctx, c, token)

			if tt.shouldFail {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			} else {
				// Should fail at JWKS retrieval step, not issuer validation
				require.Error(t, err)
				assert.Contains(t, err.Error(), "JWKS retrieval failed")
			}
		})
	}
}

// Test timeout behavior
func TestJWTAuthentication_Timeout(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	// Create a context that's already cancelled to simulate timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			"headers": map[string]string{
				"Authorization": "Bearer any.jwt.token",
			},
		},
	)
	c.SetRequest(c.Request().WithContext(ctx))

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication failed")
}

// Test that proper JWT format reaches JWKS validation step
func TestJWTAuthentication_ReachesJWKSValidation(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Create a properly formatted JWT (will fail at JWKS step since we don't have a real JWKS server)
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"iss": "https://example.com",
		"sub": "test-user",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	headerBytes, _ := json.Marshal(header)
	payloadBytes, _ := json.Marshal(payload)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := headerB64 + "." + payloadB64 + ".valid-signature-would-go-here"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			"headers": map[string]string{
				"Authorization": "Bearer " + token,
			},
		},
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		// This should not be reached due to JWKS failure
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)
	// Should fail at JWKS retrieval step, proving the two-step validation works
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication failed")
}

func TestValidateJWT_SuccessfulValidation(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Create test JWK and private key
	testJWK, privateKey, err := createTestJWK()
	require.NoError(t, err)

	// Create JWK set
	keySet := jwk.NewSet()
	keySet.AddKey(testJWK)

	// Mock GetJWKS to return our test key set
	mockGetJWKS(keySet, nil)

	defer restoreGetJWKS()

	// Create a valid signed JWT
	token, err := createSignedJWT("https://example.com", "test-user", privateKey)
	require.NoError(t, err)

	ctx := context.Background()
	c := createMockEchoContext()
	err = validateJWT(ctx, c, token)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", c.Get(h.JWTIssuer))
	assert.Equal(t, "test-user", c.Get(h.JWTSubject))
}

func TestValidateJWT_JWKSFailure(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Mock GetJWKS to return an error
	mockGetJWKS(nil, assert.AnError)

	defer restoreGetJWKS()

	// Create a basic JWT (doesn't need to be properly signed since JWKS will fail)
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"iss": "https://example.com",
		"sub": "test-user",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	headerBytes, _ := json.Marshal(header)
	payloadBytes, _ := json.Marshal(payload)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := headerB64 + "." + payloadB64 + ".dummy-signature"

	ctx := context.Background()
	c := createMockEchoContext()
	err := validateJWT(ctx, c, token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWKS retrieval failed")
}

func TestValidateJWT_InvalidSignature(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Create test JWK with different key than used for signing
	testJWK, _, err := createTestJWK()
	require.NoError(t, err)

	// Create JWK set
	keySet := jwk.NewSet()
	keySet.AddKey(testJWK)

	// Mock GetJWKS to return our test key set
	mockGetJWKS(keySet, nil)

	defer restoreGetJWKS()

	// Create another private key for signing (different from the JWK)
	differentPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create a JWT signed with different key
	token, err := createSignedJWT("https://example.com", "test-user", differentPrivateKey)
	require.NoError(t, err)

	ctx := context.Background()
	c := createMockEchoContext()
	err = validateJWT(ctx, c, token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT validation failed")
}

func TestJWTAuthentication_SuccessfulFlow(t *testing.T) {
	service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	defer service.ClearTestFeatureFlags()

	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Create test JWK and private key
	testJWK, privateKey, err := createTestJWK()
	require.NoError(t, err)

	// Create JWK set
	keySet := jwk.NewSet()
	keySet.AddKey(testJWK)

	// Mock GetJWKS to return our test key set
	mockGetJWKS(keySet, nil)

	defer restoreGetJWKS()

	// Create a valid signed JWT
	token, err := createSignedJWT("https://example.com", "test-user", privateKey)
	require.NoError(t, err)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{
			"headers": map[string]string{
				"Authorization": "Bearer " + token,
			},
		},
	)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())

	// Verify JWT claims were stored
	assert.Equal(t, "https://example.com", c.Get(h.JWTIssuer))
	assert.Equal(t, "test-user", c.Get(h.JWTSubject))
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	// Create test JWK and private key
	testJWK, privateKey, err := createTestJWK()
	require.NoError(t, err)

	// Create JWK set
	keySet := jwk.NewSet()
	keySet.AddKey(testJWK)

	// Mock GetJWKS to return our test key set
	mockGetJWKS(keySet, nil)

	defer restoreGetJWKS()

	// Create an expired JWT using the helper function
	// First, let's create a token with proper time claims, then modify the signed version
	now := time.Now()

	// Build token manually with expired time
	token, err := jwt.NewBuilder().
		Issuer("https://example.com").
		Subject("test-user").
		IssuedAt(now.Add(-2 * time.Hour)).
		Expiration(now.Add(-time.Hour)). // Expired 1 hour ago
		Build()
	require.NoError(t, err)

	// Create a private key for signing with the same key ID as the JWK
	signingKey, err := jwk.FromRaw(privateKey)
	require.NoError(t, err)

	signingKey.Set(jwk.KeyIDKey, "test-key-id")
	signingKey.Set(jwk.AlgorithmKey, jwa.RS256)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signingKey))
	require.NoError(t, err)

	ctx := context.Background()
	c := createMockEchoContext()
	err = validateJWT(ctx, c, string(signed))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT validation failed")
}
