package marketplace

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestBearerTokenUnmarshalling tests that upon receiving a properly structured response from the marketplace,
// the unmarshalling process succeeds.
func TestBearerTokenUnmarshalling(t *testing.T) {
	accessTokenValue := "test"
	expirationTimestamp := 1609455600 // 2021-01-01T00:00:00

	jsonText := fmt.Sprintf(`{"access_token": "%s", "expiration": %d}`, accessTokenValue, expirationTimestamp)

	readCloser := io.NopCloser(strings.NewReader(jsonText))
	fakeMarketplaceResponse := http.Response{Body: readCloser}

	token, err := DecodeMarketplaceTokenFromResponse(&fakeMarketplaceResponse)
	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}

	if accessTokenValue != *token.Token {
		t.Errorf("want %s, got %s", accessTokenValue, *token.Token)
	}

	if expirationTimestamp != *token.Expiration {
		t.Errorf("want %d, got %d", expirationTimestamp, token.Expiration)
	}
}

// TestInvalidJsonPassed tests that when an invalid JSON is given, the decoding function returns an error.
func TestInvalidJsonPassed(t *testing.T) {
	jsonText := `{"access_token":"abcde", "expiration"}`

	readCloser := io.NopCloser(strings.NewReader(jsonText))
	fakeMarketplaceResponse := http.Response{Body: readCloser}

	_, err := DecodeMarketplaceTokenFromResponse(&fakeMarketplaceResponse)

	if err == nil {
		t.Errorf("want error, got none")
	}
}

// TestInvalidStructurePassed tests that an error is thrown when an unexpected JSON structure is sent by the
// marketplace. For an "unexpected structure" we mean a structure that doesn't contain the access token or
// the expiration timestamp of it.
func TestInvalidStructurePassed(t *testing.T) {
	jsonText := `{"hello": "world"}`

	readCloser := io.NopCloser(strings.NewReader(jsonText))
	fakeMarketplaceResponse := http.Response{Body: readCloser}

	_, err := DecodeMarketplaceTokenFromResponse(&fakeMarketplaceResponse)

	if err == nil {
		t.Errorf("want error, got none")
	}
}
