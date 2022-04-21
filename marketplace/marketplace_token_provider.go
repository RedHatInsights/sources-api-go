package marketplace

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
)

// GetHttpClient variable that holds the function which returns an HttpClient. This allows us to set up in runtime
// which http client we want for the "GetToken" function, and allows us to mock it easily.
var GetHttpClient func() HttpClient

// GetHttpClientStdlib returns a "http.Client" with a timeout of 10 seconds.
func GetHttpClientStdlib() HttpClient {
	return &http.Client{Timeout: 10 * time.Second}
}

// MarketplaceTokenProvider is a type that satisfies the "TokenProvider" interface. The aim is to abstract away the
// injection of this dependency on other code, which will make testing easier.
type MarketplaceTokenProvider struct {
	ApiKey *string
}

// RequestToken sends a request to the marketplace to request a bearer token.
func (mtp *MarketplaceTokenProvider) RequestToken() (*BearerToken, error) {
	// Reference docs for the request: https://marketplace.redhat.com/en-us/documentation/api-authentication
	data := url.Values{}
	data.Set("apikey", *mtp.ApiKey)
	data.Set("grant_type", "urn:ibm:params:oauth:grant-type:apikey")

	request, err := http.NewRequest(
		"POST",
		config.Get().MarketplaceHost+"/api-security/om-auth/cloud/token",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		logging.Log.Errorf(`error setting up the marketplace token request: %s`, err)

		return nil, errors.New("could not properly prepare to reach the marketplace")
	}

	// Set the proper headers to accept JSON, and let the server know we're sending urlencoded data.
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		logging.Log.Errorf(`error sending the marketplace token request: %s`, err)

		return nil, errors.New("could not reach the marketplace")
	}

	if response.StatusCode != http.StatusOK {
		logging.Log.Errorf("unexpected status code received from the marketplace: %d. Response body: %s", response.StatusCode, response.Body)

		return nil, errors.New("unexpected response received from the marketplace")
	}

	return DecodeMarketplaceTokenFromResponse(response)
}
