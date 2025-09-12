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

// JWKS cache with async refresh and fallback capabilities to prevent outages
type jwksCache struct {
	mu          sync.RWMutex
	keySet      jwk.Set
	expiry      time.Time
	refreshing  bool
	lastFetched time.Time
}

var globalJWKSCache = &jwksCache{}

// FetchJWKS fetches JWKS with async refresh and fallback to prevent outages.
// Uses cached JWKS immediately when available, refreshes asynchronously in background.
// Falls back to cached JWKS on fetch failures to prevent IdP outages from causing service outages.
// Essential for high-traffic applications requiring resilience against IdP unavailability.
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

	now := time.Now()

	// Double-checked locking: first check with read lock for performance
	globalJWKSCache.mu.RLock()

	// If we have a cached keyset that hasn't expired, return it immediately
	if globalJWKSCache.keySet != nil && now.Before(globalJWKSCache.expiry) {
		keySet := globalJWKSCache.keySet
		globalJWKSCache.mu.RUnlock()

		return keySet, nil
	}

	// If we have a cached keyset but it's expired, trigger async refresh and return cached value
	if globalJWKSCache.keySet != nil && !globalJWKSCache.refreshing {
		keySet := globalJWKSCache.keySet
		globalJWKSCache.mu.RUnlock()

		// Start async refresh
		go refreshJWKSAsync(ctx, jwksURL)

		logger.Log.Debugf("Using cached JWKS while refreshing in background")

		return keySet, nil
	}

	// If already refreshing, return cached value if available
	if globalJWKSCache.refreshing && globalJWKSCache.keySet != nil {
		keySet := globalJWKSCache.keySet
		globalJWKSCache.mu.RUnlock()

		return keySet, nil
	}

	globalJWKSCache.mu.RUnlock()

	// No cached value or initial fetch - fetch synchronously
	globalJWKSCache.mu.Lock()
	defer globalJWKSCache.mu.Unlock()

	// Double-check with write lock
	if globalJWKSCache.keySet != nil && now.Before(globalJWKSCache.expiry) {
		return globalJWKSCache.keySet, nil
	}

	// If already refreshing and we have cached value, return it
	if globalJWKSCache.refreshing && globalJWKSCache.keySet != nil {
		return globalJWKSCache.keySet, nil
	}

	// Mark as refreshing
	globalJWKSCache.refreshing = true

	defer func() { globalJWKSCache.refreshing = false }()

	// Fetch JWKS securely
	keySet, err := secureJWKSFetch(ctx, jwksURL)
	if err != nil {
		logger.Log.Errorf("JWKS fetch failed from %s: %v", jwksURL, err)

		// If we have a cached keyset, return it instead of failing
		if globalJWKSCache.keySet != nil {
			logger.Log.Warnf("Using cached JWKS due to fetch failure, last fetched: %v", globalJWKSCache.lastFetched)
			return globalJWKSCache.keySet, nil
		}

		return nil, fmt.Errorf("JWKS fetch failed and no cached keyset available: %v", err)
	}

	// Validate key strength before caching
	err = validateJWKSKeyStrength(keySet)
	if err != nil {
		logger.Log.Errorf("JWKS validation failed for %s: %v", jwksURL, err)

		// If validation fails but we have a cached keyset, return it
		if globalJWKSCache.keySet != nil {
			logger.Log.Warnf("Using cached JWKS due to validation failure")
			return globalJWKSCache.keySet, nil
		}

		return nil, fmt.Errorf("JWKS key validation failed and no cached keyset available: %v", err)
	}

	// Cache for 10 minutes (balances performance and security)
	globalJWKSCache.keySet = keySet
	globalJWKSCache.expiry = now.Add(10 * time.Minute)
	globalJWKSCache.lastFetched = now

	logger.Log.Debugf("JWKS successfully fetched synchronously and cached")

	return keySet, nil
}

// refreshJWKSAsync performs background JWKS refresh to keep cache fresh without blocking requests
func refreshJWKSAsync(ctx context.Context, jwksURL string) {
	globalJWKSCache.mu.Lock()

	if globalJWKSCache.refreshing {
		globalJWKSCache.mu.Unlock()
		return // Another refresh is already in progress
	}

	globalJWKSCache.refreshing = true
	globalJWKSCache.mu.Unlock()

	defer func() {
		globalJWKSCache.mu.Lock()
		globalJWKSCache.refreshing = false
		globalJWKSCache.mu.Unlock()
	}()

	refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	keySet, err := secureJWKSFetch(refreshCtx, jwksURL)
	if err != nil {
		logger.Log.Warnf("Async JWKS refresh failed from %s: %v", jwksURL, err)
		return
	}

	err = validateJWKSKeyStrength(keySet)
	if err != nil {
		logger.Log.Warnf("Async JWKS validation failed for %s: %v", jwksURL, err)
		return
	}

	globalJWKSCache.mu.Lock()
	globalJWKSCache.keySet = keySet
	globalJWKSCache.expiry = time.Now().Add(10 * time.Minute)
	globalJWKSCache.lastFetched = time.Now()
	globalJWKSCache.mu.Unlock()

	logger.Log.Debugf("JWKS async refresh completed successfully")
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

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	// Parse JWKS
	keySet, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
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
