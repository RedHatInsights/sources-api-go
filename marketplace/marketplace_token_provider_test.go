package marketplace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
)

// fakeApiKey is a fake API key just for the tests.
var fakeApiKey = "fakeApiKey"

// TestNotReachingMarketplace tests that an error is returned when an error occurs within the HTTP Client.
func TestNotReachingMarketplace(t *testing.T) {
	// Set an invalid host as the marketplace host to force an error in the http.Client
	config.Get().MarketplaceHost = "invalidhost"

	marketplaceTokenProvider = &MarketplaceTokenProvider{ApiKey: &fakeApiKey}
	_, err := marketplaceTokenProvider.RequestToken()

	if err == nil {
		t.Errorf("want error, got none")
	}

	want := "could not reach the marketplace"
	if err.Error() != want {
		t.Errorf("want %s, got %s", want, err)
	}
}

// TestInvalidStatusCodeReturnsError checks that an error is returned when a non 200 status code is returned on the
// response from the marketplace.
func TestInvalidStatusCodeReturnsError(t *testing.T) {
	// Set up a fake test server which returns a "Bad Request" status code, instead of the "OK" that the function under
	// test expects.
	server := httptest.NewServer(
		http.HandlerFunc(
			func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusBadRequest)
			},
		),
	)
	defer server.Close()

	config.Get().MarketplaceHost = server.URL

	marketplaceTokenProvider = &MarketplaceTokenProvider{ApiKey: &fakeApiKey}
	_, err := marketplaceTokenProvider.RequestToken()

	if err == nil {
		t.Errorf("want error, got none")
	}

	want := "unexpected response received from the marketplace"
	if err.Error() != want {
		t.Errorf("want %s, got %s", want, err)
	}
}
