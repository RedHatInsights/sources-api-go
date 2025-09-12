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

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
)

// DefaultJWKSDiscoverer is the production implementation of JWKSDiscoverer
type DefaultJWKSDiscoverer struct{}

// Discover implements the JWKSDiscoverer interface
func (d *DefaultJWKSDiscoverer) Discover(ctx context.Context) (string, error) {
	return discoverJWKSURL(ctx)
}

// discoverJWKSURL discovers the JWKS URL from the issuer's OIDC discovery endpoint
func discoverJWKSURL(ctx context.Context) (string, error) {
	issuer := config.Get().JWTIssuer

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

	logger.Log.Infof("JWKS URL discovery successful: %s", jwksURL)

	return jwksURL, nil
}

// buildDiscoveryURL constructs and validates the OIDC discovery URL from the issuer
func buildDiscoveryURL(issuer string) (string, error) {
	// Validate issuer is configured
	if issuer == "" {
		return "", fmt.Errorf("JWT issuer not configured")
	}

	// Validate issuer URL
	parsedURL, err := url.Parse(issuer)
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL: %v", err)
	}

	// Ensure HTTPS for production, allow HTTP for localhost in tests
	isTestEnv := os.Getenv("GO_ENV") == "test" || strings.Contains(os.Args[0], ".test")
	isLocalHTTP := isTestEnv && parsedURL.Scheme == "http" && (parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1")

	if parsedURL.Scheme != "https" && !isLocalHTTP {
		return "", fmt.Errorf("OIDC discovery URL must be HTTPS")
	}

	// Construct OIDC discovery URL according to RFC 8414
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	return discoveryURL, nil
}

// performDiscovery fetches the OIDC discovery document and returns the JWKS URL
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
		return "", fmt.Errorf("failed to read OIDC discovery response: %w", err)
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
