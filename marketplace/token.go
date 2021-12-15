package marketplace

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BearerToken represents the bearer token sent by the marketplace, and includes the Unix timestamp of the time when
// it expires.
type BearerToken struct {
	Expiration int    `json:"expiration"`
	Token      string `json:"access_token"`
}

// DecodeMarketplaceTokenFromResponse decodes the bearer token and the expiration timestamp from the received
// response.
func DecodeMarketplaceTokenFromResponse(response *http.Response) (*BearerToken, error) {
	token := BearerToken{}

	err := json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return nil, err
	}

	if token.Expiration == 0 || token.Token == "" {
		return BearerToken{}, fmt.Errorf("unexpected JSON structure received from the marketplace")
	}

	return &token, nil
}
