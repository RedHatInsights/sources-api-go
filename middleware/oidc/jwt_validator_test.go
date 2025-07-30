package oidc

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJWT_Success(t *testing.T) {
	privateKey, publicKeyPEM, err := GenerateTestKeyPair()
	require.NoError(t, err)

	config := &JWTConfig{
		PublicKey: publicKeyPEM,
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	claims := jwt.MapClaims{
		"sub": "test-user",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	tokenString, err := CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	token, err := ValidateJWT(tokenString, config)

	require.NoError(t, err)
	assert.True(t, token.Valid)

	tokenClaims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, "test-user", tokenClaims["sub"])
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	privateKey, publicKeyPEM, err := GenerateTestKeyPair()
	require.NoError(t, err)

	config := &JWTConfig{
		PublicKey: publicKeyPEM,
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	claims := jwt.MapClaims{
		"sub": "test-user",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}

	tokenString, err := CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestValidateJWT_InvalidIssuer(t *testing.T) {
	privateKey, publicKeyPEM, err := GenerateTestKeyPair()
	require.NoError(t, err)

	config := &JWTConfig{
		PublicKey: publicKeyPEM,
		Issuer:    "expected-issuer",
		Audience:  "test-audience",
	}

	claims := jwt.MapClaims{
		"sub": "test-user",
		"iss": "wrong-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	tokenString, err := CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issuer")
}

func TestValidateJWT_InvalidAudience(t *testing.T) {
	privateKey, publicKeyPEM, err := GenerateTestKeyPair()
	require.NoError(t, err)

	config := &JWTConfig{
		PublicKey: publicKeyPEM,
		Issuer:    "test-issuer",
		Audience:  "expected-audience",
	}

	claims := jwt.MapClaims{
		"sub": "test-user",
		"iss": "test-issuer",
		"aud": "wrong-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	tokenString, err := CreateTestJWT(privateKey, claims)
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid audience")
}

func TestValidateJWT_NoPublicKey(t *testing.T) {
	config := &JWTConfig{
		PublicKey: "",
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	_, err := ValidateJWT("any-token", config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PUBLIC_KEY not configured")
}

func TestExtractUserFromJWT_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "test-user-123",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := &jwt.Token{Claims: claims}

	userID, err := ExtractUserFromJWT(token)

	require.NoError(t, err)
	assert.Equal(t, "test-user-123", userID)
}

func TestExtractUserFromJWT_MissingSubject(t *testing.T) {
	claims := jwt.MapClaims{
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := &jwt.Token{Claims: claims}

	_, err := ExtractUserFromJWT(token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid subject in token")
}

func TestExtractUserFromJWT_EmptySubject(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := &jwt.Token{Claims: claims}

	_, err := ExtractUserFromJWT(token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid subject in token")
}

func TestParseRSAPublicKey_InvalidPEM(t *testing.T) {
	invalidPEM := "not-a-valid-pem-block"

	_, err := parseRSAPublicKey(invalidPEM)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse PEM block")
}
