package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	FeatureFlagOIDCAuth = "sources-api.oidc-auth.enabled"
)

// JWTAuthentication is an Echo middleware that validates JWT tokens and extracts user identity.
//
// This middleware implements OpenID Connect (OIDC) JWT authentication with comprehensive security
// measures. It validates tokens against configured JWT issuers using JWKS for signature verification.
//
// Authentication Flow:
//  1. Check feature flag - skip if OIDC authentication is disabled
//  2. Extract JWT token from Authorization header (Bearer token)
//  3. If no token present, continue to next middleware (allowing other auth methods)
//  4. Validate token using two-step process:
//     a. Parse token and verify issuer is whitelisted
//     b. Verify signature against JWKS and validate claims
//  5. On success: store issuer/subject in context and continue
//  6. On failure: return HTTP 401 Unauthorized
//
// Security Features:
//   - Two-step validation (issuer check before expensive JWKS operation)
//   - JWKS URL discovery with caching (1-hour TTL, stale-while-revalidate)
//   - Signature verification
//   - Claims validation (exp, sub, iat required)
//   - Clock skew tolerance (30 seconds)
//   - Timeout protection (10 seconds per validation)
//   - Fails fast on JWKS retrieval errors
//
// Configuration:
//   - Feature flag: "sources-api.oidc-auth.enabled"
//   - JWT_ISSUER environment variable (required when feature enabled)
//   - Validates issuer configuration at startup
//
// Context Values Set:
//   - h.JWTIssuer: verified JWT issuer
//   - h.JWTSubject: verified JWT subject
//
// Error Handling:
//   - Returns 401 with error document on validation failure
//   - Logs validation failures at debug level
//   - Logs successful authentications at debug level
//   - Continues to next middleware if no token (non-blocking)
func JWTAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Feature flag check
			if !service.FeatureEnabled(FeatureFlagOIDCAuth) {
				return next(c)
			}

			// Extract JWT from request
			token := extractJWT(c.Request())
			if token == "" {
				return next(c) // No JWT, continue to other auth
			}

			// Validate JWT with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
			defer cancel()

			err := validateJWT(ctx, c, token)
			if err != nil {
				c.Logger().Debugf("JWT validation failed: %v", err)
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication failed", "401"))
			}

			return next(c)
		}
	}
}

func extractJWT(r *http.Request) string {
	authHeader := r.Header.Get(h.Authorization)
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		return token
	}

	return ""
}

// validateJWT validates JWT using a two-step process and sets context values:
// 1. Parse token and check if its issuer is whitelisted
// 2. Validate token claims and verify its signature with JWKS
func validateJWT(ctx context.Context, c echo.Context, token string) error {
	// Step 1: Parse token with verify and validate flags disabled to extract the issuer
	unverifiedToken, err := jwt.Parse([]byte(token),
		jwt.WithVerify(false),   // Skip signature verification - it will be done in step 2
		jwt.WithValidate(false), // Skip claims validation - it will be done in step 2
	)
	if err != nil {
		return fmt.Errorf("JWT parsing failed: %v", err)
	}

	// Reject tokens with empty or missing issuer or subject
	if unverifiedToken.Issuer() == "" {
		return fmt.Errorf("invalid JWT issuer: missing issuer")
	}

	if unverifiedToken.Subject() == "" {
		return fmt.Errorf("invalid JWT subject: missing subject")
	}

	// Validate issuer is whitelisted
	expectedIssuer := config.Get().JWTIssuer
	if unverifiedToken.Issuer() != expectedIssuer {
		return fmt.Errorf("invalid JWT issuer: expected %s, got %s", expectedIssuer, unverifiedToken.Issuer())
	}

	// Step 2: Validate token claims and verify its signature with JWKS
	jwks, err := GetJWKS(ctx, unverifiedToken.Issuer())
	if err != nil {
		return fmt.Errorf("JWKS retrieval failed: %v", err)
	}

	verifiedToken, err := jwt.Parse([]byte(token),
		jwt.WithVerify(true),
		jwt.WithKeySet(jwks),
		jwt.WithValidate(true),
		jwt.WithRequiredClaim("exp"),
		jwt.WithRequiredClaim("sub"),
		jwt.WithRequiredClaim("iat"),
		jwt.WithAcceptableSkew(30*time.Second), // Allow 30s clock drift for time claims
	)
	if err != nil {
		return fmt.Errorf("JWT validation failed: %v", err)
	}

	// Set context values for further use in the authorization process
	c.Set(h.JWTIssuer, verifiedToken.Issuer())
	c.Set(h.JWTSubject, verifiedToken.Subject())

	c.Logger().Debugf("JWT authentication successful for issuer: %s, subject: %s", verifiedToken.Issuer(), verifiedToken.Subject())

	return nil
}
