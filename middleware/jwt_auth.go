package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	FeatureFlagOIDCAuth = "sources-api.oidc-auth.enabled"
)

// JWTAuthentication validates JWT tokens using JWKS and extracts user identity.
//
// Flow: Extracts JWT from context → Validates signature with JWKS → Checks claims → Sets subject
// Returns 401 on validation failure, continues to other auth methods if no token present.
// Controlled by feature flag "sources-api.oidc-auth.enabled".
//
// Security: JWKS caching, 30s clock skew tolerance, subject length limits, timeout protection.
func JWTAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Feature flag check
			if !service.FeatureEnabled(FeatureFlagOIDCAuth) {
				return next(c)
			}

			// Extract token from context (already processed by ParseHeaders)
			token, ok := c.Get(h.JWTToken).(string)
			if !ok || token == "" {
				return next(c) // No token, continue to other auth
			}

			// Validate JWT with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
			defer cancel()

			subject, err := validateJWTToken(ctx, token)
			if err != nil {
				c.Logger().Debugf("JWT validation failed: %v", err)
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication failed", "401"))
			}

			// Store JWT subject
			c.Set(h.JWTSubject, subject)
			c.Logger().Debugf("JWT authentication successful for subject: %s", subject)

			return next(c)
		}
	}
}

// validateJWTToken validates JWT using JWKS
func validateJWTToken(ctx context.Context, tokenString string) (string, error) {
	// Fetch JWKS with caching
	keySet, err := FetchJWKS(ctx)
	if err != nil {
		return "", fmt.Errorf("JWKS fetch failed")
	}

	// Parse and validate token with enhanced validation
	token, err := jwt.Parse([]byte(tokenString),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithRequiredClaim("exp"),
		jwt.WithRequiredClaim("sub"),
		jwt.WithRequiredClaim("iat"),
		jwt.WithAcceptableSkew(30*time.Second), // Allow 30s clock drift for time claims
	)
	if err != nil {
		return "", fmt.Errorf("token validation failed")
	}

	// Validate algorithm is in allowlist
	err = validateJWTAlgorithm(token)
	if err != nil {
		return "", err
	}

	// Extract and validate subject
	subject := token.Subject()

	err = validateJWTSubject(subject)
	if err != nil {
		return "", err
	}

	return subject, nil
}

// validateJWTAlgorithm validates that the JWT uses an allowed signing algorithm
func validateJWTAlgorithm(token jwt.Token) error {
	alg := token.PrivateClaims()["alg"]
	if algStr, ok := alg.(string); ok {
		switch jwa.SignatureAlgorithm(algStr) {
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.ES256, jwa.ES384, jwa.ES512:
			// Allowed algorithms
			return nil
		case jwa.ES256K, jwa.EdDSA, jwa.HS256, jwa.HS384, jwa.HS512, jwa.NoSignature, jwa.PS256, jwa.PS384, jwa.PS512:
			// Explicitly rejected algorithms
			return fmt.Errorf("unsupported algorithm: %s", algStr)
		default:
			return fmt.Errorf("unsupported algorithm: %s", algStr)
		}
	}

	return nil
}

// validateJWTSubject validates the JWT subject claim
func validateJWTSubject(subject string) error {
	if subject == "" {
		return fmt.Errorf("missing subject")
	}

	// Validate subject length to prevent memory exhaustion attacks
	if len(subject) > 256 {
		return fmt.Errorf("subject too long: %d bytes (max 256)", len(subject))
	}

	return nil
}
