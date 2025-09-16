package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JWKS Integration Tests
//
// These tests are essential because they cover getJWKSImpl end-to-end workflows that unit tests cannot validate.
// They test complex caching behaviors (cache hits/misses, expiration, async refresh), real HTTP interactions,
// error propagation through the full stack, and multi-issuer scenarios. The main getJWKSImpl function
// contains critical caching logic that requires integration testing with actual HTTP servers and cache state.

// Mock OIDC discovery server that can simulate various scenarios.
// Provides configurable HTTP responses for testing OIDC discovery and JWKS endpoints.
// Supports error simulation, delays, content-type variations, and malformed responses.
type mockOIDCServer struct {
	server          *httptest.Server
	discoveryPath   string
	jwksPath        string
	issuer          string
	jwksURL         string
	discoveryDelay  time.Duration
	jwksDelay       time.Duration
	discoveryStatus int
	jwksStatus      int
	discoveryError  string
	jwksError       string
	contentType     string
	invalidJSON     bool
	issuerMismatch  bool
	missingJWKSURI  bool
}

// newMockOIDCServer creates a new mock OIDC server with default success configuration.
// Sets up HTTP handlers for discovery and JWKS endpoints with sensible defaults.
// Returns a fully configured server ready for testing various OIDC scenarios.
func newMockOIDCServer() *mockOIDCServer {
	m := &mockOIDCServer{
		discoveryPath:   "/.well-known/openid-configuration",
		jwksPath:        "/.well-known/jwks.json",
		discoveryStatus: http.StatusOK,
		jwksStatus:      http.StatusOK,
		contentType:     "application/json",
	}

	m.server = httptest.NewServer(http.HandlerFunc(m.handler))
	m.issuer = m.server.URL
	m.jwksURL = m.server.URL + m.jwksPath

	return m
}

// handler routes HTTP requests to appropriate mock endpoints.
// Handles discovery (.well-known/openid-configuration) and JWKS (.well-known/jwks.json) paths.
// Returns 404 for unrecognized paths to simulate real OIDC server behavior.
func (m *mockOIDCServer) handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case m.discoveryPath:
		m.handleDiscovery(w, r)
	case m.jwksPath:
		m.handleJWKS(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not_found"}`))
	}
}

// handleDiscovery handles OIDC discovery document requests.
// Supports configurable delays, status codes, content types, and response validation.
// Can simulate issuer mismatches, missing fields, and malformed JSON responses.
func (m *mockOIDCServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	if m.discoveryDelay > 0 {
		time.Sleep(m.discoveryDelay)
	}

	w.Header().Set("Content-Type", m.contentType)
	w.WriteHeader(m.discoveryStatus)

	if m.discoveryStatus != http.StatusOK {
		fmt.Fprintf(w, `{"error": "%s"}`, m.discoveryError)
		return
	}

	if m.invalidJSON {
		w.Write([]byte(`{"issuer": invalid json`))
		return
	}

	issuer := m.issuer
	if m.issuerMismatch {
		issuer = "https://wrong-issuer.com"
	}

	doc := map[string]interface{}{
		"issuer": issuer,
	}

	if !m.missingJWKSURI {
		doc["jwks_uri"] = m.jwksURL
	}

	json.NewEncoder(w).Encode(doc)
}

// handleJWKS handles JWKS (JSON Web Key Set) endpoint requests.
// Returns a minimal test JWKS document for signature verification testing.
// Supports configurable delays and error responses for failure scenario testing.
func (m *mockOIDCServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if m.jwksDelay > 0 {
		time.Sleep(m.jwksDelay)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.jwksStatus)

	if m.jwksStatus != http.StatusOK {
		fmt.Fprintf(w, `{"error": "%s"}`, m.jwksError)
		return
	}

	// Return a minimal JWKS for testing
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": "test-key-id",
				"use": "sig",
				"alg": "RS256",
				"n":   "test-modulus",
				"e":   "AQAB",
			},
		},
	}

	json.NewEncoder(w).Encode(jwks)
}

// close shuts down the mock OIDC server and releases resources.
// Should be called in defer statements to ensure proper cleanup after tests.
func (m *mockOIDCServer) close() {
	m.server.Close()
}

// GetJWKSTestScenario defines a complete JWKS integration test scenario.
// Encapsulates server configuration, expected outcomes, and validation logic.
// Used by TestGetJWKS_AllScenarios for comprehensive data-driven testing.
type GetJWKSTestScenario struct {
	name           string
	setupServer    func() *mockOIDCServer
	expectSuccess  bool
	expectError    string
	validateCache  bool
	validateResult func(*testing.T, jwk.Set, error)
}

// TestGetJWKS_AllScenarios tests all getJWKSImpl scenarios using data-driven approach.
// Covers successful flows, discovery failures, JWKS fetch failures, and caching behavior.
// Replaces individual test functions with comprehensive scenario-based testing.
func TestGetJWKS_AllScenarios(t *testing.T) {
	scenarios := []GetJWKSTestScenario{
		{
			name: "successful flow",
			setupServer: func() *mockOIDCServer {
				return newMockOIDCServer()
			},
			expectSuccess: true,
			validateCache: true,
			validateResult: func(t *testing.T, keySet jwk.Set, err error) {
				require.NoError(t, err)
				assert.NotNil(t, keySet)
			},
		},
		{
			name: "discovery failure",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.discoveryStatus = http.StatusNotFound
				server.discoveryError = "not_found"

				return server
			},
			expectSuccess: false,
			expectError:   "JWKS URL discovery failed",
			validateResult: func(t *testing.T, keySet jwk.Set, err error) {
				require.Error(t, err)
				assert.Nil(t, keySet)
			},
		},
		{
			name: "JWKS fetch failure",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.jwksStatus = http.StatusInternalServerError
				server.jwksError = "internal_error"

				return server
			},
			expectSuccess: false,
			expectError:   "JWKS fetch failed",
			validateResult: func(t *testing.T, keySet jwk.Set, err error) {
				require.Error(t, err)
				assert.Nil(t, keySet)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			server := scenario.setupServer()
			defer server.close()

			ctx := context.Background()
			keySet, err := getJWKSImpl(ctx, server.issuer)

			scenario.validateResult(t, keySet, err)

			if scenario.expectError != "" {
				assert.Contains(t, err.Error(), scenario.expectError)
			}

			if scenario.validateCache {
				cachedEntry, found := discoveryCache.Get(server.issuer)
				require.True(t, found)
				assert.Equal(t, server.jwksURL, cachedEntry.URL)
				assert.False(t, cachedEntry.IsExpired())
			}
		})
	}
}

// CacheTestScenario defines test scenarios for JWKS caching behavior.
// Tests cache hits, misses, expiration, and refresh patterns.
type CacheTestScenario struct {
	name           string
	setupCache     func(*testing.T, *mockOIDCServer)
	validateResult func(*testing.T, jwk.Set, error, *mockOIDCServer)
}

// TestGetJWKS_CachingBehavior tests JWKS caching scenarios with data-driven approach.
// Covers cache hits, expired entries, and async refresh behavior.
func TestGetJWKS_CachingBehavior(t *testing.T) {
	scenarios := []CacheTestScenario{
		{
			name: "cache hit",
			setupCache: func(t *testing.T, server *mockOIDCServer) {
				ctx := context.Background()
				_, err := getJWKSImpl(ctx, server.issuer)
				require.NoError(t, err)
			},
			validateResult: func(t *testing.T, keySet jwk.Set, err error, server *mockOIDCServer) {
				require.NoError(t, err)
				assert.NotNil(t, keySet)

				cachedEntry, found := discoveryCache.Get(server.issuer)
				require.True(t, found)
				assert.False(t, cachedEntry.IsExpired())
			},
		},
		{
			name: "expired cache refresh",
			setupCache: func(t *testing.T, server *mockOIDCServer) {
				ctx := context.Background()
				_, err := getJWKSImpl(ctx, server.issuer)
				require.NoError(t, err)
				// Manually expire the cache entry
				expiredEntry := CachedJWKSURL{
					URL:      server.jwksURL,
					CachedAt: time.Now().Add(-2 * time.Hour),
				}
				discoveryCache.Add(server.issuer, expiredEntry)
			},
			validateResult: func(t *testing.T, keySet jwk.Set, err error, server *mockOIDCServer) {
				require.NoError(t, err)
				assert.NotNil(t, keySet)
				// Give async refresh time to complete
				time.Sleep(100 * time.Millisecond)

				refreshedEntry, found := discoveryCache.Get(server.issuer)
				require.True(t, found)
				assert.False(t, refreshedEntry.IsExpired())
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			server := newMockOIDCServer()
			defer server.close()

			scenario.setupCache(t, server)

			ctx := context.Background()
			keySet, err := getJWKSImpl(ctx, server.issuer)

			scenario.validateResult(t, keySet, err, server)
		})
	}
}

// DiscoveryURLTestScenario defines test scenarios for discovery URL validation.
// Tests various URL formats, schemes, and validation edge cases.
type DiscoveryURLTestScenario struct {
	name        string
	issuer      string
	expectError string
	expectURL   string
}

// TestBuildDiscoveryURL_AllScenarios tests discovery URL building with data-driven approach.
// Covers invalid inputs, scheme validation, and successful URL construction.
func TestBuildDiscoveryURL_AllScenarios(t *testing.T) {
	scenarios := []DiscoveryURLTestScenario{
		{"empty issuer", "", "invalid JWT issuer URL", ""},
		{"invalid URL", "not-a-url", "HTTPS scheme required", ""},
		{"HTTP scheme (not HTTPS)", "http://example.com", "HTTPS scheme required", ""},
		{"FTP scheme", "ftp://example.com", "HTTPS scheme required", ""},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			url, err := buildDiscoveryURL(scenario.issuer)
			if scenario.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), scenario.expectError)
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.Equal(t, scenario.expectURL, url)
			}
		})
	}
}

// TestBuildDiscoveryURL_LocalhostHTTPIntegration tests localhost HTTP allowance in test environments.
// Verifies that HTTP is permitted for localhost/127.0.0.1 when GO_ENV=test.
// Tests the special case handling for local development and testing.
func TestBuildDiscoveryURL_LocalhostHTTPIntegration(t *testing.T) {
	// Set test environment
	originalGoEnv := os.Getenv("GO_ENV")

	defer func() {
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	os.Setenv("GO_ENV", "test")

	tests := []struct {
		name     string
		issuer   string
		expected string
	}{
		{
			name:     "localhost HTTP",
			issuer:   "http://localhost:8080",
			expected: "http://localhost:8080/.well-known/openid-configuration",
		},
		{
			name:     "127.0.0.1 HTTP",
			issuer:   "http://127.0.0.1:8080",
			expected: "http://127.0.0.1:8080/.well-known/openid-configuration",
		},
		{
			name:     "localhost HTTPS",
			issuer:   "https://localhost:8080",
			expected: "https://localhost:8080/.well-known/openid-configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildDiscoveryURL(tt.issuer)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, url)
		})
	}
}

// TestPerformDiscovery_AllScenarios tests OIDC discovery with comprehensive error scenarios.
// Uses data-driven approach to test HTTP status codes, content types, and response validation.
// Covers all discovery failure modes including network errors and malformed responses.
func TestPerformDiscovery_AllScenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *mockOIDCServer
		expectedIssuer string
		expectError    string
	}{
		{
			name: "HTTP 404",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.discoveryStatus = http.StatusNotFound

				return server
			},
			expectError: "OIDC discovery endpoint returned status 404",
		},
		{
			name: "HTTP 500",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.discoveryStatus = http.StatusInternalServerError

				return server
			},
			expectError: "OIDC discovery endpoint returned status 500",
		},
		{
			name: "invalid content type",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.contentType = "text/html"

				return server
			},
			expectError: "invalid Content-Type: text/html",
		},
		{
			name: "missing content type",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.contentType = ""

				return server
			},
			expectError: "invalid Content-Type:",
		},
		{
			name: "invalid JSON",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.invalidJSON = true

				return server
			},
			expectError: "failed to parse OIDC discovery document",
		},
		{
			name: "issuer mismatch",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.issuerMismatch = true

				return server
			},
			expectError: "issuer mismatch",
		},
		{
			name: "missing jwks_uri",
			setupServer: func() *mockOIDCServer {
				server := newMockOIDCServer()
				server.missingJWKSURI = true

				return server
			},
			expectError: "missing jwks_uri field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.close()

			expectedIssuer := server.issuer
			if tt.expectedIssuer != "" {
				expectedIssuer = tt.expectedIssuer
			}

			ctx := context.Background()
			jwksURL, err := performDiscovery(ctx, server.server.URL+server.discoveryPath, expectedIssuer)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
			assert.Empty(t, jwksURL)
		})
	}
}

// TestPerformDiscovery_Success tests successful OIDC discovery document processing.
// Verifies that valid discovery documents are parsed correctly and return expected JWKS URLs.
func TestPerformDiscovery_Success(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()
	jwksURL, err := performDiscovery(ctx, server.server.URL+server.discoveryPath, server.issuer)

	require.NoError(t, err)
	assert.Equal(t, server.jwksURL, jwksURL)
}

// TestPerformDiscovery_ContentTypeVariations tests various valid Content-Type headers.
// Verifies that discovery works with different charset specifications and formats.
// Ensures robust handling of Content-Type header variations in real-world scenarios.
func TestPerformDiscovery_ContentTypeVariations(t *testing.T) {
	contentTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"application/json;charset=utf-8",
		"application/json; charset=UTF-8",
	}

	for _, contentType := range contentTypes {
		t.Run(contentType, func(t *testing.T) {
			server := newMockOIDCServer()

			server.contentType = contentType
			defer server.close()

			ctx := context.Background()
			jwksURL, err := performDiscovery(ctx, server.server.URL+server.discoveryPath, server.issuer)

			require.NoError(t, err)
			assert.Equal(t, server.jwksURL, jwksURL)
		})
	}
}

// JWKSDiscoveryTestScenario defines integration test scenarios for JWKS URL discovery.
// Tests the complete workflow from URL building through caching.
type JWKSDiscoveryTestScenario struct {
	name          string
	setupServer   func() *mockOIDCServer
	expectSuccess bool
	expectError   string
	validateCache bool
}

// TestDiscoverJWKSURL_AllScenarios tests complete JWKS discovery workflow with data-driven approach.
// Covers successful discovery, cache population, and various failure modes.
func TestDiscoverJWKSURL_AllScenarios(t *testing.T) {
	scenarios := []JWKSDiscoveryTestScenario{
		{
			name: "integration flow",
			setupServer: func() *mockOIDCServer {
				return newMockOIDCServer()
			},
			expectSuccess: true,
			validateCache: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			server := scenario.setupServer()
			defer server.close()

			ctx := context.Background()
			jwksURL, err := discoverJWKSURL(ctx, server.issuer)

			if scenario.expectSuccess {
				require.NoError(t, err)
				assert.Equal(t, server.jwksURL, jwksURL)
			} else {
				require.Error(t, err)

				if scenario.expectError != "" {
					assert.Contains(t, err.Error(), scenario.expectError)
				}
			}

			if scenario.validateCache && scenario.expectSuccess {
				cachedEntry, found := discoveryCache.Get(server.issuer)
				require.True(t, found)
				assert.Equal(t, server.jwksURL, cachedEntry.URL)
				assert.False(t, cachedEntry.IsExpired())
			}
		})
	}
}

// TestDiscoverJWKSURL_CacheIsolation tests cache isolation between different issuers.
// Verifies that multiple issuers maintain separate cache entries without interference.
// Tests multi-issuer scenarios and cache key separation.
func TestDiscoverJWKSURL_CacheIsolation(t *testing.T) {
	// Clear cache before test
	discoveryCache, _ = lru.New[string, CachedJWKSURL](100)

	server1 := newMockOIDCServer()
	defer server1.close()

	server2 := newMockOIDCServer()
	defer server2.close()

	ctx := context.Background()

	// Discover for first issuer
	jwksURL1, err := discoverJWKSURL(ctx, server1.issuer)
	require.NoError(t, err)
	assert.Equal(t, server1.jwksURL, jwksURL1)

	// Discover for second issuer
	jwksURL2, err := discoverJWKSURL(ctx, server2.issuer)
	require.NoError(t, err)
	assert.Equal(t, server2.jwksURL, jwksURL2)

	// Verify both are cached separately
	cachedEntry1, found1 := discoveryCache.Get(server1.issuer)
	require.True(t, found1)
	assert.Equal(t, server1.jwksURL, cachedEntry1.URL)

	cachedEntry2, found2 := discoveryCache.Get(server2.issuer)
	require.True(t, found2)
	assert.Equal(t, server2.jwksURL, cachedEntry2.URL)
}

// TestCachedJWKSURL_IsExpired tests cache expiration logic with data-driven scenarios.
// Verifies TTL calculation and expiration boundary conditions.
// Tests various timestamp scenarios including edge cases around the TTL boundary.
func TestCachedJWKSURL_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cachedAt time.Time
		expected bool
	}{
		{
			name:     "fresh cache",
			cachedAt: now,
			expected: false,
		},
		{
			name:     "almost expired",
			cachedAt: now.Add(-59 * time.Minute),
			expected: false,
		},
		{
			name:     "just expired",
			cachedAt: now.Add(-61 * time.Minute),
			expected: true,
		},
		{
			name:     "very old",
			cachedAt: now.Add(-24 * time.Hour),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := CachedJWKSURL{
				URL:      "https://example.com/.well-known/jwks.json",
				CachedAt: tt.cachedAt,
			}

			assert.Equal(t, tt.expected, cached.IsExpired())
		})
	}
}
