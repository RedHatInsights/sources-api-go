package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	MaxJWKSSize = 32768 // 32KB max JWKS response
)

// Simple JWKS cache - reduces load on the JWKS endpoint
type jwksCache struct {
	mu     sync.RWMutex
	keySet jwk.Set
	expiry time.Time
}

var globalJWKSCache = &jwksCache{}

// FetchJWKS fetches JWKS with intelligent caching to optimize performance and reliability.
// Caches valid JWKS for 10 minutes to reduce load on identity providers while ensuring fresh keys.
// Includes cache invalidation on errors to prevent poisoned cache states and comprehensive
// security validation before caching. Essential for high-traffic applications with frequent JWT validation.
func FetchJWKS(ctx context.Context) (jwk.Set, error) {
	cfg := config.Get()
	jwksURL := cfg.JWKSUrl

	// Validate JWKS URL is configured and HTTPS
	if jwksURL == "" {
		return nil, fmt.Errorf("JWKS URL not configured")
	}

	// Allow HTTP URLs only for localhost/127.0.0.1 in test environment
	isTestEnv := os.Getenv("GO_ENV") == "test" || strings.Contains(os.Args[0], ".test")
	isLocalHTTP := isTestEnv && strings.HasPrefix(jwksURL, "http://") && (strings.Contains(jwksURL, "localhost") || strings.Contains(jwksURL, "127.0.0.1"))

	if !strings.HasPrefix(jwksURL, "https://") && !isLocalHTTP {
		return nil, fmt.Errorf("JWKS URL must be HTTPS")
	}

	// Double-checked locking: first check with read lock for performance
	globalJWKSCache.mu.RLock()

	if globalJWKSCache.keySet != nil && time.Now().Before(globalJWKSCache.expiry) {
		keySet := globalJWKSCache.keySet
		globalJWKSCache.mu.RUnlock()

		return keySet, nil
	}

	globalJWKSCache.mu.RUnlock()

	// Cache expired or missing, acquire write lock
	globalJWKSCache.mu.Lock()
	defer globalJWKSCache.mu.Unlock()

	// Double-check with write lock: another goroutine might have updated the cache
	if globalJWKSCache.keySet != nil && time.Now().Before(globalJWKSCache.expiry) {
		return globalJWKSCache.keySet, nil
	}

	// Fetch JWKS securely
	keySet, err := secureJWKSFetch(ctx, jwksURL)
	if err != nil {
		// Invalidate cache on error to prevent poisoning
		globalJWKSCache.keySet = nil
		globalJWKSCache.expiry = time.Time{}

		logger.Log.Errorf("JWKS fetch failed from %s: %v", jwksURL, err)

		return nil, fmt.Errorf("JWKS fetch failed: %v", err)
	}

	// Validate key strength before caching
	err = validateJWKSKeyStrength(keySet)
	if err != nil {
		// Invalidate cache on validation failure
		globalJWKSCache.keySet = nil
		globalJWKSCache.expiry = time.Time{}

		logger.Log.Errorf("JWKS validation failed for %s: %v", jwksURL, err)

		return nil, fmt.Errorf("JWKS key validation failed: %v", err)
	}

	// Cache for 10 minutes (balances performance and security)
	globalJWKSCache.keySet = keySet
	globalJWKSCache.expiry = time.Now().Add(10 * time.Minute)

	return keySet, nil
}

// secureJWKSFetch performs secure JWKS fetching with comprehensive safety controls.
// Implements multiple defenses against attacks: HTTP timeouts, status validation,
// Content-Type verification, response size limits, and key count restrictions.
// These protections prevent DoS attacks, SSRF exploitation, and malicious JWKS responses.
func secureJWKSFetch(ctx context.Context, jwksURL string) (jwk.Set, error) {
	// Create HTTP client with timeout (shorter than context timeout)
	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Validate HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	// Validate Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, fmt.Errorf("JWKS endpoint returned invalid Content-Type: %s", contentType)
	}

	// Read response with size limit (ignore Content-Length as it's spoofable)
	limitedReader := io.LimitReader(resp.Body, MaxJWKSSize+1) // +1 to detect oversized

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	// Check actual size
	if len(body) > MaxJWKSSize {
		return nil, fmt.Errorf("JWKS response too large: %d bytes", len(body))
	}

	// Pre-validate key count before parsing (quick check)
	if strings.Count(string(body), `"kid"`) > 10 {
		return nil, fmt.Errorf("too many keys in JWKS")
	}

	// Parse JWKS
	keySet, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Final validation
	if keySet.Len() > 10 {
		return nil, fmt.Errorf("too many keys in JWKS: %d", keySet.Len())
	}

	return keySet, nil
}

// validateJWKSKeyStrength validates RSA key strength in JWKS to prevent cryptographic attacks.
// Ensures all RSA keys meet minimum 2048-bit strength requirements and that at least one
// valid RSA key exists. Weak keys (< 2048 bits) are vulnerable to factorization attacks
// and should be rejected to maintain security standards.
func validateJWKSKeyStrength(keySet jwk.Set) error {
	rsaKeyCount := 0

	for i := 0; i < keySet.Len(); i++ {
		key, ok := keySet.Key(i)
		if !ok {
			continue
		}

		// Check key type
		keyType := key.KeyType()
		if keyType != jwa.RSA {
			continue // Skip non-RSA keys
		}

		rsaKeyCount++

		// Extract RSA key and check bit size
		var rawKey interface{}

		err := key.Raw(&rawKey)
		if err != nil {
			return fmt.Errorf("failed to extract raw key: %v", err)
		}

		rsaKey, ok := rawKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("key is not RSA public key")
		}

		// Validate minimum key size (2048 bits)
		if rsaKey.N.BitLen() < 2048 {
			return fmt.Errorf("RSA key too weak: %d bits (minimum 2048)", rsaKey.N.BitLen())
		}
	}

	// Ensure at least one valid RSA key exists
	if rsaKeyCount == 0 {
		return fmt.Errorf("JWKS contains no valid RSA keys")
	}

	return nil
}
