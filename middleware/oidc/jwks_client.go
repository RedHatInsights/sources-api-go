package oidc

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JWKSClient handles fetching and caching of JWKS keys
type JWKSClient struct {
	jwksURL    string
	cache      jwk.Set
	cacheMutex sync.RWMutex
	cacheTime  time.Time
	cacheTTL   time.Duration
	httpClient *http.Client
}

// NewJWKSClient creates a new JWKS client with HTTPS validation
func NewJWKSClient(jwksURL string) (*JWKSClient, error) {
	// Validate that JWKS URL uses HTTPS
	if !strings.HasPrefix(jwksURL, "https://") {
		return nil, fmt.Errorf("JWKS URL must use HTTPS: %s", jwksURL)
	}

	return &JWKSClient{
		jwksURL:    jwksURL,
		cacheTTL:   5 * time.Minute, // Default cache TTL, can be overridden by headers
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// GetKeySet returns the cached JWKS or fetches it if expired/missing
func (c *JWKSClient) GetKeySet(ctx context.Context) (jwk.Set, error) {
	c.cacheMutex.RLock()

	if c.cache != nil && time.Since(c.cacheTime) < c.cacheTTL {
		defer c.cacheMutex.RUnlock()
		return c.cache, nil
	}

	c.cacheMutex.RUnlock()

	// Need to fetch/refresh
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Double-check pattern - another goroutine might have updated
	if c.cache != nil && time.Since(c.cacheTime) < c.cacheTTL {
		return c.cache, nil
	}

	// Fetch from JWKS URL with retry logic
	keySet, err := c.fetchWithRetry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %v", c.jwksURL, err)
	}

	c.cache = keySet
	c.cacheTime = time.Now()

	return keySet, nil
}

// fetchWithRetry implements retry logic with exponential backoff
func (c *JWKSClient) fetchWithRetry(ctx context.Context) (jwk.Set, error) {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Create request to get cache headers
		req, err := http.NewRequestWithContext(ctx, "GET", c.jwksURL, nil)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, err
			}

			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, err
			}

			continue
		}

		// Parse cache headers
		c.parseCacheHeaders(resp.Header)
		resp.Body.Close()

		// Now use jwk.Fetch for actual fetching
		keySet, err := jwk.Fetch(ctx, c.jwksURL)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, err
			}

			continue
		}

		return keySet, nil
	}

	return nil, fmt.Errorf("exhausted all retry attempts")
}

// parseCacheHeaders extracts cache TTL from HTTP headers
func (c *JWKSClient) parseCacheHeaders(headers http.Header) {
	// Check Cache-Control header
	if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		if maxAge := extractMaxAge(cacheControl); maxAge > 0 {
			c.cacheTTL = time.Duration(maxAge) * time.Second
			return
		}
	}

	// Check Expires header
	if expires := headers.Get("Expires"); expires != "" {
		expireTime, err := http.ParseTime(expires)
		if err == nil {
			ttl := time.Until(expireTime)
			if ttl > 0 {
				c.cacheTTL = ttl
				return
			}
		}
	}

	// Default TTL if no cache headers
	c.cacheTTL = 5 * time.Minute
}

// extractMaxAge parses max-age from Cache-Control header
func extractMaxAge(cacheControl string) int {
	parts := strings.Split(cacheControl, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			maxAge, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil {
				return maxAge
			}
		}
	}

	return 0
}

// GetPublicKey retrieves a specific public key by key ID (kid)
func (c *JWKSClient) GetPublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	keySet, err := c.GetKeySet(ctx)
	if err != nil {
		return nil, err
	}

	key, found := keySet.LookupKeyID(keyID)
	if !found {
		// Key not found - try refreshing JWKS to handle key rotation
		c.cacheMutex.Lock()
		c.cache = nil // Force refresh
		c.cacheMutex.Unlock()

		// Try fetching fresh JWKS
		keySet, err = c.GetKeySet(ctx)
		if err != nil {
			return nil, err
		}

		// Try again with fresh key set
		key, found = keySet.LookupKeyID(keyID)
		if !found {
			return nil, fmt.Errorf("key with ID %s not found in JWKS after refresh", keyID)
		}
	}

	var publicKey rsa.PublicKey

	err = key.Raw(&publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to extract RSA public key: %v", err)
	}

	return &publicKey, nil
}

// ValidateTokenWithJWKS validates a JWT token using JWKS key discovery
func ValidateTokenWithJWKS(ctx context.Context, tokenString string, jwksClient *JWKSClient, issuer, audience string) (jwt.Token, error) {
	// Parse JWS to get the key ID from header
	parsedJWS, err := jws.Parse([]byte(tokenString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWS: %v", err)
	}

	// Get key ID from JWS header - REQUIRED for JWKS
	keyIDInterface, ok := parsedJWS.Signatures()[0].ProtectedHeaders().Get("kid")
	if !ok {
		return nil, fmt.Errorf("JWT missing required 'kid' (key ID) in header")
	}

	keyID, ok := keyIDInterface.(string)
	if !ok || keyID == "" {
		return nil, fmt.Errorf("JWT 'kid' (key ID) must be a non-empty string")
	}

	// Get the public key for the specified key ID
	publicKey, err := jwksClient.GetPublicKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key for kid %s: %v", keyID, err)
	}

	// Verify the token with the public key
	verifiedToken, err := jwt.Parse([]byte(tokenString),
		jwt.WithKey(jwa.RS256, publicKey),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %v", err)
	}

	// Validate issuer if specified
	if issuer != "" {
		if tokenIssuer := verifiedToken.Issuer(); tokenIssuer != issuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %s", issuer, tokenIssuer)
		}
	}

	// Validate audience if specified
	if audience != "" {
		audiences := verifiedToken.Audience()
		found := false

		for _, aud := range audiences {
			if aud == audience {
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("invalid audience: expected %s, got %v", audience, audiences)
		}
	}

	return verifiedToken, nil
}

// ExtractUserFromJWKSToken extracts user information from a JWKS-validated token
func ExtractUserFromJWKSToken(token jwt.Token) (string, error) {
	subject := token.Subject()
	if subject == "" {
		return "", fmt.Errorf("missing or invalid subject in token")
	}

	return subject, nil
}
