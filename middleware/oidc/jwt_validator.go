package oidc

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds configuration for JWT validation
type JWTConfig struct {
	PublicKey string
	Issuer    string
	Audience  string
	JWKSUrl   string
}

// LoadJWTConfigFromGlobal loads JWT configuration from the global config
func LoadJWTConfigFromGlobal() *JWTConfig {
	cfg := config.Get()

	return &JWTConfig{
		PublicKey: cfg.JWTPublicKey,
		Issuer:    cfg.JWTIssuer,
		Audience:  cfg.JWTAudience,
		JWKSUrl:   cfg.JWTJWKSUrl,
	}
}

// parseRSAPublicKey parses RSA public key from PEM format
func parseRSAPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return rsaPub, nil
}

// ValidateJWT validates a JWT token and returns the parsed token
func ValidateJWT(tokenString string, config *JWTConfig) (*jwt.Token, error) {
	if config.PublicKey == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY not configured")
	}

	publicKey, err := parseRSAPublicKey(config.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check issuer if configured
	if config.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != config.Issuer {
			return nil, fmt.Errorf("invalid issuer")
		}
	}

	// Check audience if configured
	if config.Audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != config.Audience {
			return nil, fmt.Errorf("invalid audience")
		}
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token has expired")
		}
	}

	return token, nil
}

// ExtractUserFromJWT extracts user information from JWT token
func ExtractUserFromJWT(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Extract subject as user ID
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing or invalid subject in token")
	}

	return sub, nil
}

// ValidateJWTWithConfig validates a JWT token using either static key or JWKS
func ValidateJWTWithConfig(tokenString string, config *JWTConfig) (string, error) {
	// If JWKS URL is configured, use JWKS validation
	if config.JWKSUrl != "" {
		return validateJWTWithJWKS(tokenString, config)
	}

	// Otherwise use static key validation
	token, err := ValidateJWT(tokenString, config)
	if err != nil {
		return "", err
	}

	return ExtractUserFromJWT(token)
}

// validateJWTWithJWKS validates JWT using JWKS discovery
func validateJWTWithJWKS(tokenString string, config *JWTConfig) (string, error) {
	jwksClient, err := NewJWKSClient(config.JWKSUrl)
	if err != nil {
		return "", fmt.Errorf("failed to create JWKS client: %v", err)
	}

	ctx := context.Background()

	token, err := ValidateTokenWithJWKS(ctx, tokenString, jwksClient, config.Issuer, config.Audience)
	if err != nil {
		return "", err
	}

	return ExtractUserFromJWKSToken(token)
}
