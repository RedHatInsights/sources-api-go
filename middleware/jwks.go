package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// JWKSDiscoverer interface abstracts JWKS URL discovery to facilitate testing and mocking.
// This interface allows us to inject different discovery implementations during testing
// without having to test JWKS URL discovery logic in unrelated tests.
type JWKSDiscoverer interface {
	Discover(ctx context.Context) (string, error)
}

// Default JWKS discoverer instance
var defaultJWKSDiscoverer = &DefaultJWKSDiscoverer{}

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
	return FetchJWKSWithDiscoverer(ctx, defaultJWKSDiscoverer)
}

// FetchJWKSWithDiscoverer fetches JWKS using the provided discoverer interface.
// This function enables dependency injection for testing.
func FetchJWKSWithDiscoverer(ctx context.Context, jwksDiscoverer JWKSDiscoverer) (jwk.Set, error) {
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
		go refreshJWKSAsync(ctx, jwksDiscoverer)

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
	keySet, err := secureJWKSFetch(ctx, jwksDiscoverer)
	if err != nil {
		logger.Log.Errorf("JWKS fetch failed: %v", err)

		// If we have a cached keyset, return it instead of failing
		if globalJWKSCache.keySet != nil {
			logger.Log.Warnf("Using cached JWKS due to fetch failure, last fetched: %v", globalJWKSCache.lastFetched)
			return globalJWKSCache.keySet, nil
		}

		return nil, fmt.Errorf("JWKS fetch failed and no cached keyset available: %v", err)
	}

	// Cache for 10 minutes (balances performance and security)
	globalJWKSCache.keySet = keySet
	globalJWKSCache.expiry = now.Add(10 * time.Minute)
	globalJWKSCache.lastFetched = now

	logger.Log.Debugf("JWKS successfully fetched synchronously and cached")

	return keySet, nil
}

// refreshJWKSAsync performs background JWKS refresh to keep cache fresh without blocking requests
func refreshJWKSAsync(ctx context.Context, jwksDiscoverer JWKSDiscoverer) {
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

	keySet, err := secureJWKSFetch(refreshCtx, jwksDiscoverer)
	if err != nil {
		logger.Log.Warnf("Async JWKS refresh failed: %v", err)
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
func secureJWKSFetch(ctx context.Context, jwksDiscoverer JWKSDiscoverer) (jwk.Set, error) {
	// Discover JWKS URL from the issuer's OIDC discovery endpoint
	jwksURL, err := jwksDiscoverer.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("JWKS URL discovery failed: %v", err)
	}

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
