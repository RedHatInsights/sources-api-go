package marketplace

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetToken sends a request to the marketplace to request a bearer token.
func GetToken(marketplaceHost string, apiKey string) (BearerToken, error) {
	// Reference docs for the request: https://marketplace.redhat.com/en-us/documentation/api-authentication
	data := url.Values{}
	data.Set("apikey", apiKey)
	data.Set("grant_type", "urn:ibm:params:oauth:grant-type:apikey")

	client := http.Client{}
	request, err := http.NewRequest(
		"POST",
		marketplaceHost+"/api-security/om-auth/cloud/token",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		return BearerToken{}, fmt.Errorf("could not create the request object: %s", err)
	}

	// Set the proper headers to accept JSON, and let the server know we're sending urlencoded data.
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(request)
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not perform the request to the marketplace: %s", err)
	}

	if response.StatusCode != http.StatusOK {
		return BearerToken{}, fmt.Errorf("unexpected status code received from the marketplace: %d", response.StatusCode)
	}

	return DecodeMarketplaceTokenFromResponse(response)
}
