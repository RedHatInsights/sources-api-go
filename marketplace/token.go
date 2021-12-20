package marketplace

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BearerToken represents the bearer token sent by the marketplace, and includes the Unix timestamp of the time when
// it expires.
type BearerToken struct {
	Expiration *int64  `json:"expiration"`
	Token      *string `json:"access_token"`
}

// MarshalBinary implements the "BinaryMarshaller" interface to easily marshal the struct when using the Redis client.
func (bt BearerToken) MarshalBinary() (data []byte, err error) {
	return json.Marshal(bt)
}

// String pretty prints the struct. Required to avoid printing the actual pointer addresses.
func (bt *BearerToken) String() string {
	return fmt.Sprintf(`BearerToken{Expiration:%d, Token:"%s"}`, *bt.Expiration, *bt.Token)
}

// DecodeMarketplaceTokenFromResponse decodes the bearer token and the expiration timestamp from the received
// response.
func DecodeMarketplaceTokenFromResponse(response *http.Response) (*BearerToken, error) {
	token := BearerToken{}

	err := json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return nil, err
	}

	err = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close the marketplace JWT token response's body: %s", err)
	}

	if token.Expiration == nil || token.Token == nil {
		return nil, fmt.Errorf("unexpected JSON structure received from the marketplace")
	}

	return &token, nil
}
