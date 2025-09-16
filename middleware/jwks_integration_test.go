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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock OIDC discovery server that can simulate various scenarios
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

func (m *mockOIDCServer) close() {
	m.server.Close()
}

func TestGetJWKS_SuccessfulFlow(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()

	// First call should discover JWKS URL
	keySet, err := getJWKSImpl(ctx, server.issuer)
	require.NoError(t, err)
	assert.NotNil(t, keySet)

	// Verify cache was populated
	cachedEntry, found := discoveryCache.Get(server.issuer)
	require.True(t, found)
	assert.Equal(t, server.jwksURL, cachedEntry.URL)
	assert.False(t, cachedEntry.IsExpired())
}

func TestGetJWKS_DiscoveryFailure(t *testing.T) {
	server := newMockOIDCServer()
	server.discoveryStatus = http.StatusNotFound

	server.discoveryError = "not_found"
	defer server.close()

	ctx := context.Background()

	keySet, err := getJWKSImpl(ctx, server.issuer)
	require.Error(t, err)
	assert.Nil(t, keySet)
	assert.Contains(t, err.Error(), "JWKS URL discovery failed")
}

func TestGetJWKS_JWKSFetchFailure(t *testing.T) {
	server := newMockOIDCServer()
	server.jwksStatus = http.StatusInternalServerError

	server.jwksError = "internal_error"
	defer server.close()

	ctx := context.Background()

	keySet, err := getJWKSImpl(ctx, server.issuer)
	require.Error(t, err)
	assert.Nil(t, keySet)
	assert.Contains(t, err.Error(), "JWKS fetch failed")
}

func TestGetJWKS_CacheHit(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()

	// First call to populate cache
	_, err := getJWKSImpl(ctx, server.issuer)
	require.NoError(t, err)

	// Verify cache entry exists
	cachedEntry, found := discoveryCache.Get(server.issuer)
	require.True(t, found)
	assert.False(t, cachedEntry.IsExpired())

	// Second call should use cached URL
	keySet, err := getJWKSImpl(ctx, server.issuer)
	require.NoError(t, err)
	assert.NotNil(t, keySet)
}

func TestGetJWKS_ExpiredCacheRefresh(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()

	// First call to populate cache
	_, err := getJWKSImpl(ctx, server.issuer)
	require.NoError(t, err)

	// Manually expire the cache entry
	expiredEntry := CachedJWKSURL{
		URL:      server.jwksURL,
		CachedAt: time.Now().Add(-2 * time.Hour),
	}
	discoveryCache.Add(server.issuer, expiredEntry)

	// Verify entry is expired
	cachedEntry, found := discoveryCache.Get(server.issuer)
	require.True(t, found)
	assert.True(t, cachedEntry.IsExpired())

	// Call should still work using stale cache and trigger async refresh
	keySet, err := getJWKSImpl(ctx, server.issuer)
	require.NoError(t, err)
	assert.NotNil(t, keySet)

	// Give async refresh time to complete
	time.Sleep(100 * time.Millisecond)

	// Cache should now be refreshed
	refreshedEntry, found := discoveryCache.Get(server.issuer)
	require.True(t, found)
	assert.False(t, refreshedEntry.IsExpired())
}

func TestBuildDiscoveryURL_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		issuer      string
		expectError string
	}{
		{
			name:        "empty issuer",
			issuer:      "",
			expectError: "invalid JWT issuer URL",
		},
		{
			name:        "invalid URL",
			issuer:      "not-a-url",
			expectError: "HTTPS scheme required",
		},
		{
			name:        "HTTP scheme (not HTTPS)",
			issuer:      "http://example.com",
			expectError: "HTTPS scheme required",
		},
		{
			name:        "FTP scheme",
			issuer:      "ftp://example.com",
			expectError: "HTTPS scheme required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildDiscoveryURL(tt.issuer)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
			assert.Empty(t, url)
		})
	}
}

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

func TestPerformDiscovery_ErrorScenarios(t *testing.T) {
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

func TestPerformDiscovery_Success(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()
	jwksURL, err := performDiscovery(ctx, server.server.URL+server.discoveryPath, server.issuer)

	require.NoError(t, err)
	assert.Equal(t, server.jwksURL, jwksURL)
}

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

func TestDiscoverJWKSURL_IntegrationFlow(t *testing.T) {
	server := newMockOIDCServer()
	defer server.close()

	ctx := context.Background()

	// Test discovery flow
	jwksURL, err := discoverJWKSURL(ctx, server.issuer)
	require.NoError(t, err)
	assert.Equal(t, server.jwksURL, jwksURL)

	// Verify cache was populated
	cachedEntry, found := discoveryCache.Get(server.issuer)
	require.True(t, found)
	assert.Equal(t, server.jwksURL, cachedEntry.URL)
	assert.False(t, cachedEntry.IsExpired())
}

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
