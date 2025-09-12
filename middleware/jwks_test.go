package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Test helper to create JWKS with many keys
func createJWKSWithManyKeys() string {
	keys := make([]*rsa.PrivateKey, 15) // More than the 10 key limit
	for i := range keys {
		keys[i] = generateTestRSAKey(2048)
	}

	jwks := createTestJWKS(keys...)

	data, err := json.Marshal(jwks)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal large JWKS: %v", err))
	}

	return string(data)
}

func TestValidateJWKSKeyStrength(t *testing.T) {
	tests := []struct {
		name    string
		keySet  jwk.Set
		wantErr bool
		errMsg  string
	}{
		{
			name:    "strong 2048-bit key",
			keySet:  createTestJWKS(generateTestRSAKey(2048)),
			wantErr: false,
		},
		{
			name:    "strong 4096-bit key",
			keySet:  createTestJWKS(generateTestRSAKey(4096)),
			wantErr: false,
		},
		{
			name:    "multiple strong keys",
			keySet:  createTestJWKS(generateTestRSAKey(2048), generateTestRSAKey(4096)),
			wantErr: false,
		},
		{
			name:    "weak 1024-bit key",
			keySet:  createTestJWKS(generateTestRSAKey(1024)),
			wantErr: true,
			errMsg:  "RSA key too weak: 1024 bits",
		},
		{
			name:    "mixed strong and weak keys",
			keySet:  createTestJWKS(generateTestRSAKey(2048), generateTestRSAKey(1024)),
			wantErr: true,
			errMsg:  "RSA key too weak: 1024 bits",
		},
		{
			name:    "empty JWKS",
			keySet:  jwk.NewSet(),
			wantErr: true,
			errMsg:  "JWKS contains no valid RSA keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWKSKeyStrength(tt.keySet)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSecureJWKSFetch(t *testing.T) {
	validJWKS := createValidJWKSJSON()
	largeJWKS := createJWKSWithManyKeys()
	oversizedBody := strings.Repeat("x", MaxJWKSSize+1000)

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
			name:         "oversized response",
			responseCode: http.StatusOK,
			contentType:  "application/json",
			body:         oversizedBody,
			wantErr:      true,
			errMsg:       "JWKS response too large:",
		},
		{
			name:         "too many keys",
			responseCode: http.StatusOK,
			contentType:  "application/json",
			body:         largeJWKS,
			wantErr:      true,
			errMsg:       "too many keys in JWKS",
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}

				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			ctx := context.Background()
			keySet, err := secureJWKSFetch(ctx, server.URL)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCallCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validJWKS))
	}))
	defer server.Close()

	// Set environment variable for JWKS URL (config reads from env)
	originalURL := os.Getenv("JWKS_URL")

	defer func() {
		if originalURL != "" {
			os.Setenv("JWKS_URL", originalURL)
		} else {
			os.Unsetenv("JWKS_URL")
		}

		config.Reset() // Reset config cache
	}()

	os.Setenv("JWKS_URL", server.URL)
	config.Reset() // Force config reload
	t.Run("successful fetch and cache", func(t *testing.T) {
		httpCallCount = 0

		resetGlobalCache()

		ctx := context.Background()

		// First call - should fetch from server
		keys1, err := FetchJWKS(ctx)
		require.NoError(t, err)
		assert.NotNil(t, keys1)
		assert.Equal(t, 1, httpCallCount)

		// Second call - should use cache
		keys2, err := FetchJWKS(ctx)
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
		_, err := FetchJWKS(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, httpCallCount)

		// Manually expire cache
		globalJWKSCache.mu.Lock()
		globalJWKSCache.expiry = time.Now().Add(-1 * time.Hour) // Expired
		globalJWKSCache.mu.Unlock()

		// Second call - with async refresh, this returns cached value immediately
		// and triggers background refresh
		_, err = FetchJWKS(ctx)
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
		jwksURL string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing JWKS URL",
			jwksURL: "",
			wantErr: true,
			errMsg:  "JWKS URL not configured",
		},
		{
			name:    "HTTP URL (not HTTPS)",
			jwksURL: "http://example.com/.well-known/jwks.json",
			wantErr: true,
			errMsg:  "JWKS URL must be HTTPS",
		},
		{
			name:    "valid HTTPS URL",
			jwksURL: "https://example.com/.well-known/jwks.json",
			wantErr: true,                // Will fail due to network, but should pass URL validation
			errMsg:  "JWKS fetch failed", // Different error - means URL validation passed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalCache()

			// Set environment variable for this test
			originalURL := os.Getenv("JWKS_URL")

			defer func() {
				if originalURL != "" {
					os.Setenv("JWKS_URL", originalURL)
				} else {
					os.Unsetenv("JWKS_URL")
				}

				config.Reset()
			}()

			if tt.jwksURL != "" {
				os.Setenv("JWKS_URL", tt.jwksURL)
			} else {
				os.Unsetenv("JWKS_URL")
			}

			config.Reset() // Force config reload

			ctx := context.Background()
			_, err := FetchJWKS(ctx)

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

	// Set environment variable for JWKS URL
	originalURL := os.Getenv("JWKS_URL")

	defer func() {
		if originalURL != "" {
			os.Setenv("JWKS_URL", originalURL)
		} else {
			os.Unsetenv("JWKS_URL")
		}

		config.Reset()
	}()

	os.Setenv("JWKS_URL", server.URL)
	config.Reset()

	t.Run("fallback to cached JWKS on fetch failure", func(t *testing.T) {
		httpCallCount = 0
		shouldFail = false

		resetGlobalCache()

		ctx := context.Background()

		// First call - should succeed and cache the result
		keys1, err := FetchJWKS(ctx)
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
		keys2, err := FetchJWKS(ctx)
		require.NoError(t, err)
		assert.NotNil(t, keys2)
		assert.Equal(t, keys1.Len(), keys2.Len())
	})

	t.Run("async refresh behavior", func(t *testing.T) {
		httpCallCount = 0
		shouldFail = false

		resetGlobalCache()

		ctx := context.Background()

		// First call - should succeed and cache the result
		keys1, err := FetchJWKS(ctx)
		require.NoError(t, err)
		assert.NotNil(t, keys1)
		assert.Equal(t, 1, httpCallCount)

		// Manually expire cache to trigger async refresh
		globalJWKSCache.mu.Lock()
		originalExpiry := globalJWKSCache.expiry
		globalJWKSCache.expiry = time.Now().Add(-1 * time.Hour)
		globalJWKSCache.mu.Unlock()

		// Second call - should return cached value immediately and trigger async refresh
		keys2, err := FetchJWKS(ctx)
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
	})
}
