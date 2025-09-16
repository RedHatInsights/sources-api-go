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

type JWTValidationResult struct {
	Issuer  string
	Subject string
}

const (
	FeatureFlagOIDCAuth = "sources-api.oidc-auth.enabled"
)

// JWTAuthentication validates JWT tokens for internal API authentication.
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
//  5. Check if issuer/subject pair is whitelisted
//  6. On success: store issuer/subject in context for PermissionCheck middleware
//  7. On failure: return HTTP 401/403
//
// Security Features:
//   - Two-step validation (issuer check before expensive JWKS operation)
//   - JWKS URL discovery with caching (1-hour TTL, stale-while-revalidate)
//   - Signature verification with JWKS
//   - Claims validation (exp, sub, iat required)
//   - Clock skew tolerance (30 seconds)
//   - Timeout protection (10 seconds per validation)
//   - Authorization check against configured issuer/subject whitelist
//
// Configuration:
//   - Feature flag: "sources-api.oidc-auth.enabled"
//   - JWT_ISSUER environment variable (required when feature enabled)
//   - AUTHORIZED_JWT_SUBJECTS environment variable (JSON array of issuer/subject pairs)
//
// Context Values Set (signals successful JWT auth to PermissionCheck):
//   - h.JWTIssuer: verified JWT issuer
//   - h.JWTSubject: verified JWT subject
//
// Error Handling:
//   - Returns 401 for invalid tokens, 403 for unauthorized subjects
//   - Logs validation failures at debug level
//   - Continues to next middleware if no token (non-blocking)
func JWTAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if JWT auth is enabled from Unleash
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

			result, err := validateJWT(ctx, c, token)
			if err != nil {
				c.Logger().Debugf("JWT validation failed: %v", err)
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication failed", "401"))
			}

			if isJWTSubjectAuthorized(result.Issuer, result.Subject) {
				// Store JWT claims in context for PermissionCheck middleware
				c.Set(h.JWTIssuer, result.Issuer)
				c.Set(h.JWTSubject, result.Subject)
				c.Logger().Debugf("JWT authentication successful for issuer: %s, subject: %s", result.Issuer, result.Subject)
			} else {
				return c.JSON(http.StatusForbidden, util.NewErrorDoc("JWT subject not authorized", "403"))
			}

			return next(c)
		}
	}
}

// extractJWT extracts the JWT token from the Authorization header.
func extractJWT(r *http.Request) string {
	authHeader := r.Header.Get(h.Authorization)
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		return token
	}

	return ""
}

// validateJWT validates JWT using a two-step process and returns verification result:
// 1. Parse JWT and check if its issuer is whitelisted
// 2. Validate JWT claims and verify its signature with JWKS
func validateJWT(ctx context.Context, c echo.Context, token string) (*JWTValidationResult, error) {
	// Step 1: Parse JWT to extract issuer (no verification yet)
	unverifiedToken, err := jwt.Parse([]byte(token),
		jwt.WithVerify(false),   // Skip signature verification - it will be done in step 2
		jwt.WithValidate(false), // Skip claims validation - it will be done in step 2
	)
	if err != nil {
		return nil, fmt.Errorf("JWT parsing failed: %v", err)
	}

	// Require issuer and subject claims
	if unverifiedToken.Issuer() == "" {
		return nil, fmt.Errorf("invalid JWT issuer: missing issuer")
	}

	if unverifiedToken.Subject() == "" {
		return nil, fmt.Errorf("invalid JWT subject: missing subject")
	}

	// Check issuer is whitelisted in the configuration
	expectedIssuer := config.Get().JWTIssuer
	if unverifiedToken.Issuer() != expectedIssuer {
		return nil, fmt.Errorf("invalid JWT issuer: expected %s, got %s", expectedIssuer, unverifiedToken.Issuer())
	}

	// Step 2: Full validation with JWKS
	jwks, err := GetJWKS(ctx, unverifiedToken.Issuer())
	if err != nil {
		return nil, fmt.Errorf("JWKS retrieval failed: %v", err)
	}

	verifiedToken, err := jwt.Parse([]byte(token),
		jwt.WithVerify(true),
		jwt.WithKeySet(jwks),
		jwt.WithValidate(true),
		jwt.WithRequiredClaim("exp"),
		jwt.WithRequiredClaim("sub"),
		jwt.WithRequiredClaim("iat"),
		jwt.WithAcceptableSkew(30*time.Second), // Allow clock drift for time claims
	)
	if err != nil {
		return nil, fmt.Errorf("JWT validation failed: %v", err)
	}

	result := &JWTValidationResult{
		Issuer:  verifiedToken.Issuer(),
		Subject: verifiedToken.Subject(),
	}

	c.Logger().Debugf("JWT validation successful for issuer: %s, subject: %s", result.Issuer, result.Subject)

	return result, nil
}

// isJWTSubjectAuthorized returns true if the given JWT issuer/subject pair is whitelisted in the configuration
func isJWTSubjectAuthorized(jwtIssuer, jwtSubject string) bool {
	for _, authorized := range config.Get().AuthorizedJWTSubjects {
		if authorized.Issuer == jwtIssuer && authorized.Subject == jwtSubject {
			return true
		}
	}

	return false
}
