package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/hashicorp/golang-lru/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// jwksCache stores JWKS documents with automatic background refresh.
// Uses jwx library's built-in caching with 1-hour refresh interval.
// Thread-safe with graceful handling of refresh failures.
var jwksCache = jwk.NewCache(context.Background())

// discoveryCache maps issuer URLs to discovered JWKS URLs with timestamps.
// LRU cache with 10 issuer capacity, 1-hour TTL, stale-while-revalidate pattern.
// Thread-safe for concurrent access.
var discoveryCache, _ = lru.New[string, CachedJWKSURL](10)

// CachedJWKSURL stores a discovered JWKS URL with timestamp for cache expiration.
// Used to implement stale-while-revalidate pattern in discoveryCache.
type CachedJWKSURL struct {
	URL      string    // The JWKS endpoint URL
	CachedAt time.Time // When this URL was discovered and cached
}

const (
	// discoveryTTL defines TTL for JWKS URL discovery cache (1 hour).
	// After expiration, triggers async refresh while serving stale data.
	discoveryTTL = time.Hour
)

// IsExpired returns true if the cached entry has exceeded discoveryTTL.
// Used for stale-while-revalidate: expired entries trigger async refresh.
func (c CachedJWKSURL) IsExpired() bool {
	return time.Since(c.CachedAt) > discoveryTTL
}

// GetJWKS retrieves JWKS for the given issuer with two-level caching.
// Function variable allows mocking in tests.
var GetJWKS = getJWKSImpl

// getJWKSImpl retrieves JWKS (JSON Web Key Set) for JWT signature verification.
//
// This function implements a sophisticated caching strategy with two levels:
// 1. JWKS URL discovery cache (1-hour TTL with stale-while-revalidate)
// 2. JWKS content cache (1-hour refresh interval with automatic background updates)
//
// Flow:
//  1. Check discovery cache for JWKS URL
//     - Cache miss: Discover JWKS URL synchronously, cache result
//     - Cache hit (fresh): Use cached JWKS URL
//     - Cache hit (expired): Use stale JWKS URL, trigger async refresh
//  2. Register JWKS URL with automatic refresh (if not already registered)
//  3. Fetch JWKS from content cache (handles background refresh automatically)
//
// Caching Benefits:
//   - Reduces latency by avoiding repeated OIDC discovery calls
//   - Improves reliability with stale-while-revalidate pattern
//   - Scales well under high load with background refresh
//   - Maintains performance during temporary issuer unavailability
//
// Error Handling:
//   - Discovery failures: Return error immediately (fail-fast)
//   - JWKS fetch failures: Return error with issuer context
//   - Async refresh failures: Log warning, continue with stale data
//
// Thread Safety:
//   - LRU cache is thread-safe for concurrent access
//   - JWX cache handles concurrent JWKS fetching
//   - Async refresh uses separate goroutine to avoid blocking
func getJWKSImpl(ctx context.Context, issuer string) (jwk.Set, error) {
	// Try to get cached JWKS URL entry
	cachedEntry, found := discoveryCache.Get(issuer)

	var jwksURL string

	if !found {
		// Cache miss - discover JWKS URL synchronously for first request
		discoveredURL, err := discoverJWKSURL(ctx, issuer)
		if err != nil {
			return nil, fmt.Errorf("JWKS URL discovery failed for issuer %s: %v", issuer, err)
		}

		jwksURL = discoveredURL
		logger.Log.Debugf("Initial JWKS URL discovery for issuer: %s -> %s", issuer, jwksURL)
	} else {
		// Cache hit - we're using the cached URL regardless of expiration
		jwksURL = cachedEntry.URL

		if !cachedEntry.IsExpired() {
			logger.Log.Debugf("Using fresh cached JWKS URL for issuer: %s -> %s", issuer, jwksURL)
		} else {
			// Cache hit but expired - use stale value and trigger async refresh
			logger.Log.Debugf("Using expired cached JWKS URL for issuer: %s -> %s (triggering refresh)", issuer, jwksURL)

			// Trigger async refresh (fire-and-forget) for future requests
			go func() { //nolint:contextcheck
				// Use context.Background() since this refresh is for future requests,
				// not the current request which is already being served with stale data
				refreshedURL, err := discoverJWKSURL(context.Background(), issuer)
				if err != nil {
					logger.Log.Warnf("Async JWKS URL refresh failed for issuer %s: %v", issuer, err)
					return // Keep using stale URL
				}

				if refreshedURL != jwksURL {
					logger.Log.Infof("JWKS URL updated for issuer %s: %s -> %s", issuer, jwksURL, refreshedURL)
				}
			}()
		}
	}

	// Register JWKS URL with 1-hour refresh interval only if not already registered
	if !jwksCache.IsRegistered(jwksURL) {
		jwksCache.Register(jwksURL, jwk.WithMinRefreshInterval(time.Hour))
		logger.Log.Debugf("Registered JWKS URL for JWKS caching: %s", jwksURL)
	}

	// Fetch JWKS from cache (handles JWKS caching and refresh automatically)
	keySet, err := jwksCache.Get(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("JWKS fetch failed for issuer %s: %v", issuer, err)
	}

	logger.Log.Debugf("JWKS successfully fetched from cache for issuer %s", issuer)

	return keySet, nil
}

// discoverJWKSURL discovers JWKS URL using OIDC discovery endpoint.
// Implements RFC 8414 with issuer validation and HTTPS enforcement.
// Caches result with timestamp for future use.
func discoverJWKSURL(ctx context.Context, issuer string) (string, error) {
	// Build and validate OIDC discovery URL
	discoveryURL, err := buildDiscoveryURL(issuer)
	if err != nil {
		return "", fmt.Errorf("OIDC discovery URL validation failed: %v", err)
	}

	// Perform JWKS URL discovery
	jwksURL, err := performDiscovery(ctx, discoveryURL, issuer)
	if err != nil {
		return "", fmt.Errorf("JWKS URL discovery failed: %v", err)
	}

	// Cache the discovered JWKS URL with timestamp
	newCacheEntry := CachedJWKSURL{
		URL:      jwksURL,
		CachedAt: time.Now(),
	}
	discoveryCache.Add(issuer, newCacheEntry)

	logger.Log.Infof("JWKS URL discovery successful: %s", jwksURL)

	return jwksURL, nil
}

// buildDiscoveryURL constructs OIDC discovery URL from issuer.
// Appends "/.well-known/openid-configuration" with HTTPS validation.
// Allows HTTP for localhost in test environments only.
func buildDiscoveryURL(issuer string) (string, error) {
	// Parse issuer URL
	parsedURL, err := url.Parse(issuer)
	if err != nil {
		return "", fmt.Errorf("invalid JWT issuer URL: %v", err)
	}

	// Ensure HTTPS for production, allow HTTP for localhost in tests
	isTestEnv := os.Getenv("GO_ENV") == "test" || strings.Contains(os.Args[0], ".test")
	isLocalHTTP := isTestEnv && parsedURL.Scheme == "http" && (parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1")

	if parsedURL.Scheme != "https" && !isLocalHTTP {
		return "", fmt.Errorf("invalid JWT issuer URL: HTTPS scheme required")
	}

	// Build OIDC discovery URL according to RFC 8414
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	return discoveryURL, nil
}

// performDiscovery fetches OIDC discovery document and extracts jwks_uri.
// Validates HTTP status, content-type, and issuer field for security.
// Uses 8-second timeout with context cancellation support.
func performDiscovery(ctx context.Context, discoveryURL, expectedIssuer string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Validate HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery endpoint returned status %d", resp.StatusCode)
	}

	// Validate Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return "", fmt.Errorf("OIDC discovery endpoint returned invalid Content-Type: %s", contentType)
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC discovery response body: %w", err)
	}

	// Parse only the fields we need from the OIDC discovery document
	var discoveryDoc struct {
		Issuer  string `json:"issuer"`
		JWKSUri string `json:"jwks_uri"`
	}

	err = json.Unmarshal(body, &discoveryDoc)
	if err != nil {
		return "", fmt.Errorf("failed to parse OIDC discovery document: %w", err)
	}

	// Validate that returned issuer matches expected issuer
	if discoveryDoc.Issuer != expectedIssuer {
		return "", fmt.Errorf("OIDC discovery document issuer mismatch: expected %s, got %s", expectedIssuer, discoveryDoc.Issuer)
	}

	if discoveryDoc.JWKSUri == "" {
		return "", fmt.Errorf("OIDC discovery document missing jwks_uri field")
	}

	return discoveryDoc.JWKSUri, nil
}
