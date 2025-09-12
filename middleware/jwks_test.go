package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock JWKS discoverer for testing
type mockJWKSDiscoverer struct {
	jwksURL string
	err     error
}

func (m *mockJWKSDiscoverer) Discover(ctx context.Context) (string, error) {
	return m.jwksURL, m.err
}

// Helper to create mock JWKS discoverer
func newMockJWKSDiscoverer(jwksURL string, err error) JWKSDiscoverer {
	return &mockJWKSDiscoverer{jwksURL: jwksURL, err: err}
}

// Test helper to reset global cache
func resetGlobalCache() {
	globalJWKSCache.mu.Lock()
	defer globalJWKSCache.mu.Unlock()

	globalJWKSCache.keySet = nil
	globalJWKSCache.expiry = time.Time{}
	globalJWKSCache.refreshing = false
	globalJWKSCache.lastFetched = time.Time{}
}

// Test helper to create RSA key with specific bit size
func generateTestRSAKey(bits int) *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(fmt.Sprintf("failed to generate %d-bit RSA key: %v", bits, err))
	}

	return key
}

// Test helper to create JWKS with given RSA keys
func createTestJWKS(keys ...*rsa.PrivateKey) jwk.Set {
	set := jwk.NewSet()
	for i, key := range keys {
		jwkKey, err := jwk.FromRaw(&key.PublicKey)
		if err != nil {
			panic(fmt.Sprintf("failed to create JWK from RSA key: %v", err))
		}

		jwkKey.Set("kid", fmt.Sprintf("key-%d", i))
		jwkKey.Set("use", "sig")
		jwkKey.Set("alg", "RS256")
		set.AddKey(jwkKey)
	}

	return set
}

// Test helper to create valid JWKS JSON
func createValidJWKSJSON() string {
	key2048 := generateTestRSAKey(2048)
	jwks := createTestJWKS(key2048)

	data, err := json.Marshal(jwks)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JWKS: %v", err))
	}

	return string(data)
}

func TestSecureJWKSFetch(t *testing.T) {
	validJWKS := createValidJWKSJSON()

	tests := []struct {
		name         string
		responseCode int
		contentType  string
		body         string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "valid JWKS response",
			responseCode: http.StatusOK,
			contentType:  "application/json",
			body:         validJWKS,
			wantErr:      false,
		},
		{
			name:         "valid JWKS with charset",
			responseCode: http.StatusOK,
			contentType:  "application/json; charset=utf-8",
			body:         validJWKS,
			wantErr:      false,
		},
		{
			name:         "404 not found",
			responseCode: http.StatusNotFound,
			contentType:  "application/json",
			body:         `{"error": "not found"}`,
			wantErr:      true,
			errMsg:       "JWKS endpoint returned status 404",
		},
		{
			name:         "500 internal error",
			responseCode: http.StatusInternalServerError,
			contentType:  "application/json",
			body:         `{"error": "internal error"}`,
			wantErr:      true,
			errMsg:       "JWKS endpoint returned status 500",
		},
		{
			name:         "wrong content type",
			responseCode: http.StatusOK,
			contentType:  "text/html",
			body:         `<html><body>Not JSON</body></html>`,
			wantErr:      true,
			errMsg:       "JWKS endpoint returned invalid Content-Type: text/html",
		},
		{
			name:         "missing content type",
			responseCode: http.StatusOK,
			contentType:  "",
			body:         validJWKS,
			wantErr:      true,
			errMsg:       "JWKS endpoint returned invalid Content-Type:",
		},
		{
			name:         "invalid JSON",
			responseCode: http.StatusOK,
			contentType:  "application/json",
			body:         `{"keys": [invalid json`,
			wantErr:      true,
			errMsg:       "failed to parse JWKS:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalCache()

			// JWKS server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}

				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			ctx := context.Background()
			mockJWKSDiscoverer := newMockJWKSDiscoverer(server.URL, nil)
			keySet, err := secureJWKSFetch(ctx, mockJWKSDiscoverer)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, keySet)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, keySet)
				assert.Positive(t, keySet.Len())
			}
		})
	}
}

func TestFetchJWKS(t *testing.T) {
	// Reset cache before each test
	resetGlobalCache()

	validJWKS := createValidJWKSJSON()
	httpCallCount := 0

	// JWKS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCallCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validJWKS))
	}))
	defer server.Close()

	mockJWKSDiscoverer := newMockJWKSDiscoverer(server.URL, nil)

	t.Run("successful fetch and cache", func(t *testing.T) {
		httpCallCount = 0

		resetGlobalCache()

		ctx := context.Background()

		// First call - should fetch from server
		keys1, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys1)
		assert.Equal(t, 1, httpCallCount)

		// Second call - should use cache
		keys2, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys2)
		assert.Equal(t, 1, httpCallCount) // No additional HTTP call

		// Keys should be the same (from cache)
		assert.Equal(t, keys1.Len(), keys2.Len())
	})

	t.Run("cache expiry", func(t *testing.T) {
		httpCallCount = 0

		resetGlobalCache()

		ctx := context.Background()

		// First call
		_, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.Equal(t, 1, httpCallCount)

		// Manually expire cache
		globalJWKSCache.mu.Lock()
		globalJWKSCache.expiry = time.Now().Add(-1 * time.Hour) // Expired
		globalJWKSCache.mu.Unlock()

		// Second call - with async refresh, this returns cached value immediately
		// and triggers background refresh
		_, err = FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)

		// Wait for async refresh to complete
		time.Sleep(200 * time.Millisecond)

		// Now the HTTP call count should be 2 due to async refresh
		assert.Equal(t, 2, httpCallCount)
	})
}

func TestFetchJWKS_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mockURL string
		mockErr error
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing JWT issuer",
			mockURL: "",
			mockErr: fmt.Errorf("OIDC discovery URL validation failed: JWT issuer not configured"),
			wantErr: true,
			errMsg:  "JWT issuer not configured",
		},
		{
			name:    "HTTP issuer (not HTTPS)",
			mockURL: "",
			mockErr: fmt.Errorf("OIDC discovery URL validation failed: OIDC discovery URL must be HTTPS"),
			wantErr: true,
			errMsg:  "OIDC discovery URL must be HTTPS",
		},
		{
			name:    "discovery network failure",
			mockURL: "",
			mockErr: fmt.Errorf("JWKS URL discovery failed: no such host"),
			wantErr: true,
			errMsg:  "JWKS URL discovery failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalCache()

			ctx := context.Background()
			// Create mock JWKS discoverer with the specified error
			mockJWKSDiscoverer := newMockJWKSDiscoverer(tt.mockURL, tt.mockErr)
			_, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFetchJWKS_AsyncRefreshAndFallback(t *testing.T) {
	resetGlobalCache()

	validJWKS := createValidJWKSJSON()
	httpCallCount := 0
	shouldFail := false

	// JWKS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCallCount++

		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validJWKS))
	}))
	defer server.Close()

	// Create mock JWKS discoverer
	mockJWKSDiscoverer := newMockJWKSDiscoverer(server.URL, nil)

	t.Run("fallback to cached JWKS on fetch failure", func(t *testing.T) {
		httpCallCount = 0
		shouldFail = false

		resetGlobalCache()

		ctx := context.Background()

		// First call - should succeed and cache the result
		keys1, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys1)
		assert.Equal(t, 1, httpCallCount)

		// Manually expire cache to trigger refresh
		globalJWKSCache.mu.Lock()
		globalJWKSCache.expiry = time.Now().Add(-1 * time.Hour)
		globalJWKSCache.mu.Unlock()

		// Make server fail
		shouldFail = true

		// Second call - should use cached value despite server failure
		keys2, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys2)
		assert.Equal(t, keys1.Len(), keys2.Len())

		// Wait for any pending async operations to complete and prevent race conditions between tests
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("async refresh behavior", func(t *testing.T) {
		httpCallCount = 0
		shouldFail = false

		resetGlobalCache()

		ctx := context.Background()

		// First call - should succeed and cache the result
		keys1, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys1)
		assert.Equal(t, 1, httpCallCount)

		// Manually expire cache to trigger async refresh
		globalJWKSCache.mu.Lock()
		originalExpiry := globalJWKSCache.expiry
		globalJWKSCache.expiry = time.Now().Add(-1 * time.Hour)
		globalJWKSCache.mu.Unlock()

		// Second call - should return cached value immediately and trigger async refresh
		keys2, err := FetchJWKSWithDiscoverer(ctx, mockJWKSDiscoverer)
		require.NoError(t, err)
		assert.NotNil(t, keys2)
		assert.Equal(t, keys1.Len(), keys2.Len())

		// Wait a bit for async refresh to complete
		time.Sleep(200 * time.Millisecond)

		// Verify that cache was refreshed (expiry should be updated)
		globalJWKSCache.mu.RLock()
		newExpiry := globalJWKSCache.expiry
		refreshing := globalJWKSCache.refreshing
		globalJWKSCache.mu.RUnlock()

		assert.True(t, newExpiry.After(originalExpiry), "Cache should be refreshed with new expiry")
		assert.False(t, refreshing, "Should not be refreshing anymore")
		assert.Equal(t, 2, httpCallCount)
	})
}
