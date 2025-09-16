package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strings"
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

// Test helpers
// JWKS mocking: Tests replace the global GetJWKS function with mockGetJWKS()
// to return controlled RSA key sets instead of fetching from real OIDC endpoints.
// Tests create matching private keys for JWT signing, enabling full validation
// of the two-step process (issuer check + signature verification).

var originalGetJWKS func(context.Context, string) (jwk.Set, error)

// mockGetJWKS replaces the global GetJWKS function with a mock that returns predetermined values.
// This allows tests to control JWKS responses without making real HTTP requests to OIDC endpoints.
// Used to simulate both successful key retrieval and JWKS server failures.
func mockGetJWKS(keySet jwk.Set, err error) {
	originalGetJWKS = GetJWKS
	GetJWKS = func(ctx context.Context, issuer string) (jwk.Set, error) {
		return keySet, err
	}
}

// restoreGetJWKS restores the original GetJWKS function after mocking.
// Should be called in defer statements to ensure proper cleanup after each test.
func restoreGetJWKS() {
	if originalGetJWKS != nil {
		GetJWKS = originalGetJWKS
		originalGetJWKS = nil
	}
}

// createTestJWK generates a test RSA key pair and creates a corresponding JWK.
// Returns both the JWK (for JWKS mocking) and private key (for JWT signing).
// The generated key uses RS256 algorithm with a fixed key ID for consistent testing.
func createTestJWK() (jwk.Key, *rsa.PrivateKey, error) {
	// Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create JWK from public key
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	key.Set(jwk.KeyIDKey, "test-key-id")
	key.Set(jwk.AlgorithmKey, jwa.RS256)
	key.Set(jwk.KeyUsageKey, "sig")

	return key, privateKey, nil
}

// createSignedJWT creates a properly signed JWT token using the provided private key.
// The token includes standard claims (iss, sub, iat, exp) and is signed with RS256.
// Used for end-to-end testing where signature verification is required.
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

	// Create signing key
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

// JWTTestScenario defines a complete JWT test scenario with all necessary configuration.
// Encapsulates feature flags, token configuration, JWKS mocking, and expected outcomes.
// Used by runJWTScenario to execute comprehensive end-to-end JWT authentication tests.
// Supports testing feature flag states, token validation, authorization, and error conditions.
type JWTTestScenario struct {
	name           string
	configIssuer   string
	tokenIssuer    string
	tokenSubject   string
	authSubjects   map[string]string // issuer -> subject for authorized pairs
	mockJWKSError  bool
	featureEnabled bool
	expectedStatus int
	expectedError  string
	shouldHaveJWT  bool // whether context should have JWT values set
}

// TestJWTAuthentication_AllScenarios tests all JWT authentication middleware scenarios.
// Uses data-driven testing to cover feature flags, token validation, JWKS failures, and authorization.
// Each scenario tests the complete middleware flow from token extraction to context setting.
// Replaces multiple individual test functions with a comprehensive test suite.
func TestJWTAuthentication_AllScenarios(t *testing.T) {
	scenarios := []JWTTestScenario{
		// Feature flag scenarios
		{
			name:           "feature flag disabled",
			featureEnabled: false,
			tokenIssuer:    "https://example.com",
			tokenSubject:   "test-user",
			expectedStatus: http.StatusOK,
			shouldHaveJWT:  false,
		},
		{
			name:           "feature enabled, no token",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "", // Explicitly no token
			tokenSubject:   "",
			expectedStatus: http.StatusOK,
			shouldHaveJWT:  false,
		},
		// Invalid token scenarios
		{
			name:           "invalid token format",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "invalid", // Mark as invalid token
			tokenSubject:   "invalid",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication failed",
			shouldHaveJWT:  false,
		},
		// Issuer validation scenarios
		{
			name:           "missing token issuer",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "",
			tokenSubject:   "test-user",
			mockJWKSError:  true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication failed",
			shouldHaveJWT:  false,
		},
		{
			name:           "issuer mismatch",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "https://wrong.com",
			tokenSubject:   "test-user",
			mockJWKSError:  true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication failed",
			shouldHaveJWT:  false,
		},
		// JWKS failure scenarios
		{
			name:           "JWKS retrieval fails",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "https://example.com",
			tokenSubject:   "test-user",
			mockJWKSError:  true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication failed",
			shouldHaveJWT:  false,
		},
		// Authorization scenarios
		{
			name:           "unauthorized subject",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "https://example.com",
			tokenSubject:   "unauthorized-user",
			authSubjects:   map[string]string{"https://example.com": "authorized-user"},
			expectedStatus: http.StatusForbidden,
			expectedError:  "JWT subject not authorized",
			shouldHaveJWT:  false,
		},
		// Success scenarios
		{
			name:           "successful authentication",
			featureEnabled: true,
			configIssuer:   "https://example.com",
			tokenIssuer:    "https://example.com",
			tokenSubject:   "test-user",
			authSubjects:   map[string]string{"https://example.com": "test-user"},
			expectedStatus: http.StatusOK,
			shouldHaveJWT:  true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runJWTScenario(t, scenario)
		})
	}
}

// setupTestEnv configures the JWT_ISSUER environment variable for testing.
// Saves the original value and provides a cleanup function to restore it.
// Forces config reset to ensure the new issuer value is loaded.
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

// createMockEchoContext creates a minimal Echo context for testing validateJWT function.
// Uses the test utilities to create a context without authentication headers.
// Primarily used for unit testing validateJWT in isolation from middleware.
func createMockEchoContext() echo.Context {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/test",
		nil,
		map[string]interface{}{},
	)

	return c
}

// createUnsignedJWT creates an unsigned JWT token with dummy signature for testing.
// Used to test token parsing and issuer validation without requiring valid signatures.
// Appends a dummy signature to bypass basic JWT format validation.
func createUnsignedJWT(issuer, subject string) string {
	now := time.Now()
	builder := jwt.NewBuilder().
		Subject(subject).
		IssuedAt(now).
		Expiration(now.Add(time.Hour))

	if issuer != "" {
		builder = builder.Issuer(issuer)
	}

	token, err := builder.Build()
	if err != nil {
		panic(err)
	}

	tokenBytes, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		panic(err)
	}

	return string(tokenBytes) + ".dummy-signature"
}

// createValidJWTWithKeys creates a properly signed JWT and mocks JWKS for complete validation.
// Generates test keys, mocks the GetJWKS function, and returns a signed token.
// Returns both the token and a cleanup function to restore the original GetJWKS.
// Used for end-to-end testing where full JWT signature verification is required.
func createValidJWTWithKeys(issuer, subject string) (string, func()) {
	testJWK, privateKey, err := createTestJWK()
	if err != nil {
		panic(err)
	}

	keySet := jwk.NewSet()
	keySet.AddKey(testJWK)
	mockGetJWKS(keySet, nil)

	token, err := createSignedJWT(issuer, subject, privateKey)
	if err != nil {
		panic(err)
	}

	return token, restoreGetJWKS
}

// assertValidateJWTSuccess verifies that validateJWT successfully processes a valid token.
// Checks that the function returns no error and extracts the correct issuer and subject.
// Used in tests where JWT validation should succeed.
func assertValidateJWTSuccess(t *testing.T, token, expectedIssuer, expectedSubject string) {
	ctx := context.Background()
	c := createMockEchoContext()
	result, err := validateJWT(ctx, c, token)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedIssuer, result.Issuer)
	assert.Equal(t, expectedSubject, result.Subject)
}

// assertValidateJWTFailure verifies that validateJWT rejects invalid tokens with expected errors.
// Ensures the function returns an error containing the specified error message.
// Used in negative test cases where JWT validation should fail.
func assertValidateJWTFailure(t *testing.T, token, expectedError string) {
	ctx := context.Background()
	c := createMockEchoContext()
	result, err := validateJWT(ctx, c, token)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), expectedError)
}

// runJWTScenario executes a complete JWT authentication test scenario.
// Handles all test setup: feature flags, config, authorization, token creation, and JWKS mocking.
// Runs the middleware and validates expected outcomes (status codes, errors, context values).
// Central execution engine that replaces repetitive test setup across multiple test functions.
func runJWTScenario(t *testing.T, scenario JWTTestScenario) {
	// Setup feature flag
	if scenario.featureEnabled {
		service.SetTestFeatureFlag(FeatureFlagOIDCAuth, true)
	} else {
		service.ClearTestFeatureFlags()
	}

	defer service.ClearTestFeatureFlags()

	// Setup config
	cleanup := setupTestEnv(t, scenario.configIssuer)
	defer cleanup()

	// Setup authorized subjects if any
	var cleanupAuth func()

	if len(scenario.authSubjects) > 0 {
		authList := make([]string, 0, len(scenario.authSubjects))
		for issuer, subject := range scenario.authSubjects {
			authList = append(authList, fmt.Sprintf(`{"issuer":"%s","subject":"%s"}`, issuer, subject))
		}

		originalValue := os.Getenv("AUTHORIZED_JWT_SUBJECTS")
		os.Setenv("AUTHORIZED_JWT_SUBJECTS", "["+strings.Join(authList, ",")+"]")

		cleanupAuth = func() {
			if originalValue != "" {
				os.Setenv("AUTHORIZED_JWT_SUBJECTS", originalValue)
			} else {
				os.Unsetenv("AUTHORIZED_JWT_SUBJECTS")
			}
		}
		defer cleanupAuth()
	}

	// Create token
	var token string

	var restoreJWKS func()

	if scenario.tokenIssuer == "" && scenario.tokenSubject == "" && !scenario.mockJWKSError {
		// No token case - leave token empty
		token = ""
	} else if scenario.tokenIssuer == "invalid" && scenario.tokenSubject == "invalid" {
		// Invalid token format
		token = "invalid.jwt.token"
	} else if scenario.mockJWKSError {
		mockGetJWKS(nil, assert.AnError)

		restoreJWKS = restoreGetJWKS
		token = createUnsignedJWT(scenario.tokenIssuer, scenario.tokenSubject)
	} else if scenario.tokenIssuer != "" && scenario.tokenSubject != "" {
		token, restoreJWKS = createValidJWTWithKeys(scenario.tokenIssuer, scenario.tokenSubject)
	} else {
		token = "invalid.jwt.token"
	}

	if restoreJWKS != nil {
		defer restoreJWKS()
	}

	// Run middleware
	var headers map[string]interface{}
	if token != "" {
		headers = map[string]interface{}{
			"headers": map[string]string{"Authorization": "Bearer " + token},
		}
	} else {
		headers = map[string]interface{}{} // No authorization header
	}

	c, rec := request.CreateTestContext(http.MethodGet, "/test", nil, headers)

	middleware := JWTAuthentication()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	err := handler(c)
	require.NoError(t, err)

	// Assert results
	assert.Equal(t, scenario.expectedStatus, rec.Code)

	if scenario.expectedError != "" {
		assert.Contains(t, rec.Body.String(), scenario.expectedError)
	}

	if scenario.shouldHaveJWT {
		assert.Equal(t, scenario.tokenIssuer, c.Get(h.JWTIssuer))
		assert.Equal(t, scenario.tokenSubject, c.Get(h.JWTSubject))
	} else {
		assert.Nil(t, c.Get(h.JWTIssuer))
		assert.Nil(t, c.Get(h.JWTSubject))
	}
}

// TestValidateJWT_AllScenarios tests validateJWT function with comprehensive input variations.
// Covers token parsing failures, issuer validation, JWKS errors, and successful validation.
// Uses data-driven testing to verify both positive and negative scenarios.
// Tests the function in isolation without full middleware context.
func TestValidateJWT_AllScenarios(t *testing.T) {
	cleanup := setupTestEnv(t, "https://example.com")
	defer cleanup()

	tests := []struct {
		name            string
		token           string
		setupJWKS       func() func() // returns cleanup function
		expectSuccess   bool
		expectError     string
		expectedIssuer  string
		expectedSubject string
	}{
		// Token parsing failures
		{"malformed token", "not.a.jwt", nil, false, "JWT parsing failed", "", ""},
		{"empty token", "", nil, false, "JWT parsing failed", "", ""},
		{"invalid base64", "invalid.base64!.token", nil, false, "JWT parsing failed", "", ""},

		// Issuer validation
		{"missing issuer", createUnsignedJWT("", "test-user"), nil, false, "missing issuer", "", ""},
		{"wrong issuer", createUnsignedJWT("https://wrong.com", "test-user"), nil, false, "expected https://example.com, got https://wrong.com", "", ""},
		{"missing subject", createUnsignedJWT("https://example.com", ""), nil, false, "missing subject", "", ""},

		// JWKS failures
		{"JWKS error", createUnsignedJWT("https://example.com", "test-user"), func() func() {
			mockGetJWKS(nil, assert.AnError)
			return restoreGetJWKS
		}, false, "JWKS retrieval failed", "", ""},

		// Success case
		{"valid token", "", func() func() {
			_, cleanup := createValidJWTWithKeys("https://example.com", "test-user")
			return cleanup
		}, true, "", "https://example.com", "test-user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			if tt.setupJWKS != nil {
				cleanup = tt.setupJWKS()
				defer cleanup()

				if tt.name == "valid token" {
					// Special case: get token from JWKS setup
					tt.token, _ = createValidJWTWithKeys("https://example.com", "test-user")
				}
			}

			if tt.expectSuccess {
				assertValidateJWTSuccess(t, tt.token, tt.expectedIssuer, tt.expectedSubject)
			} else {
				assertValidateJWTFailure(t, tt.token, tt.expectError)
			}
		})
	}
}

// TestExtractJWT tests JWT token extraction from HTTP Authorization headers.
// Verifies proper handling of Bearer tokens, malformed headers, and missing tokens.
// Tests the first step of JWT authentication - extracting tokens from requests.
func TestExtractJWT(t *testing.T) {
	tests := []struct {
		header, expect string
	}{
		{"Bearer my.jwt.token", "my.jwt.token"},
		{"Bearer   my.jwt.token  ", "my.jwt.token"},
		{"", ""},
		{"Basic dXNlcjpwYXNz", ""},
		{"Bearer", ""},
		{"Bearer ", ""},
	}

	for _, tt := range tests {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		if tt.header != "" {
			req.Header.Set("Authorization", tt.header)
		}

		assert.Equal(t, tt.expect, extractJWT(req))
	}
}

// TestIsJWTSubjectAuthorized tests the JWT subject authorization whitelist logic.
// Verifies that only issuer/subject pairs configured in AUTHORIZED_JWT_SUBJECTS are allowed.
// Tests the final authorization step after successful JWT validation.
func TestIsJWTSubjectAuthorized(t *testing.T) {
	originalValue := os.Getenv("AUTHORIZED_JWT_SUBJECTS")
	os.Setenv("AUTHORIZED_JWT_SUBJECTS", `[{"issuer":"https://example.com","subject":"user1"},{"issuer":"https://other.com","subject":"user2"}]`)

	defer func() {
		if originalValue != "" {
			os.Setenv("AUTHORIZED_JWT_SUBJECTS", originalValue)
		} else {
			os.Unsetenv("AUTHORIZED_JWT_SUBJECTS")
		}
	}()

	config.Reset()
	defer config.Reset()

	tests := []struct {
		issuer, subject string
		expected        bool
	}{
		{"https://example.com", "user1", true},
		{"https://other.com", "user2", true},
		{"https://wrong.com", "user1", false},
		{"https://example.com", "wrong-user", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, isJWTSubjectAuthorized(tt.issuer, tt.subject))
	}
}
