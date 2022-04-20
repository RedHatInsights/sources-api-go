package marketplace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/sirupsen/logrus"
)

// marketplaceTokenCacher is a variable that holds the "GetMarketplaceTokenCacher" function, or any function that is
// similar to that one. This way we can inject the "TokenCacher" dependency at runtime, which enables easier mocking
// and testing.
var marketplaceTokenCacher TokenCacher

// GetMarketplaceTokenCacher stores a function that returns a TokenCacher
var GetMarketplaceTokenCacher func(*int64) TokenCacher

// GetMarketplaceTokenCacherWithTenantId is the default implementation which returns a default "TokenCacher" instance,
// which is used in the main application.
func GetMarketplaceTokenCacherWithTenantId(tenantId *int64) TokenCacher {
	return &MarketplaceTokenCacher{TenantID: *tenantId}
}

// marketplaceTokenProvider is a variable similar to marketplaceTokenCacher, which holds the function that returns the
// required dependency.
var marketplaceTokenProvider TokenProvider

// GetMarketplaceTokenProvider stores a function that retunrs a TokenProvider
var GetMarketplaceTokenProvider func(string) TokenProvider

// GetMarketplaceTokenProviderWithApiKey is the default implementation which returns a default "TokenProvider" instance,
// which is used in the main application.
func GetMarketplaceTokenProviderWithApiKey(apiKey string) TokenProvider {
	return &MarketplaceTokenProvider{ApiKey: &apiKey}
}

// Assign the default token cacher and providers for the "SetMarketplaceTokenAuthExtraField". They can be overridden
// in runtime.
func init() {
	GetMarketplaceTokenCacher = GetMarketplaceTokenCacherWithTenantId
	GetMarketplaceTokenProvider = GetMarketplaceTokenProviderWithApiKey
}

// SetMarketplaceTokenAuthExtraField tries to put the marketplace token as a JSON string in the "auth.Extra" field
// only if the provided authentication is of the type "marketplace".
func SetMarketplaceTokenAuthExtraField(tenantId int64, auth *model.Authentication) error {
	// If the authentication isn't a "marketplace" auth, then skip getting the token
	if auth.AuthType != "marketplace-token" {
		return nil
	}

	var token *BearerToken

	// First try to fetch the token from the cache
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)
	token, err := marketplaceTokenCacher.FetchToken()

	// If it's not present, request one token to the marketplace, cache it, and assign it to the "extra" field
	// of auth
	if err != nil {
		// The Api key must be present to be able to send the request to the marketplace
		if auth.Password == nil {
			return errors.New("API key not present for the marketplace authentication")
		}

		var apiKey string
		if config.IsVaultOn() {
			apiKey = *auth.Password
		} else {
			// When using the database backed authentications we need to decrypt the API Key to be able to fetch a proper
			// token from the marketplace.
			decryptedPassword, err := util.Decrypt(*auth.Password)
			if err != nil {
				return err
			}

			apiKey = decryptedPassword
		}

		marketplaceTokenProvider = GetMarketplaceTokenProvider(apiKey)

		token, err = marketplaceTokenProvider.RequestToken()
		if err != nil {
			return fmt.Errorf("could not get token from the marketplace: %s", err)
		}

		// Cache the token. We really don't mind if we cannot properly cache it: we can request another one and
		// return the one that we got. But we log the error for traceability and future debugging.
		err = marketplaceTokenCacher.CacheToken(token)
		if err != nil {
			logger.Log.Errorf("could not cache the token in Redis: %s", err)
		}
	}

	if config.IsVaultOn() {
		if auth.Extra == nil {
			auth.Extra = make(map[string]interface{})
		}

		auth.Extra["marketplace"] = token
	} else {
		if auth.ExtraDb == nil {
			// In case there is no content in the database we can safely marshal the token and return it directly.
			auth.ExtraDb, err = json.Marshal(token)
			if err != nil {
				return err
			}
		} else {
			var extra = make(map[string]interface{})
			extra["marketplace"] = token

			// The "extra" column in the database may contain JSON already, and in that case we need to unmarshal that
			// content into the struct to make sure we don't overwrite it. However, if the existing content happens to be
			// the valid JSON "null" value, we must skip unmarshalling that to the map, since "json.Unmarshal" just turns
			// the map "nil".
			if auth.ExtraDb != nil && !bytes.Equal(auth.ExtraDb, []byte("null")) {
				err := json.Unmarshal(auth.ExtraDb, &extra)
				if err != nil {
					return err
				}
			}

			auth.ExtraDb, err = json.Marshal(extra)
			if err != nil {
				return err
			}
		}
	}

	logger.Log.Log(logrus.InfoLevel, "marketplace token included in authentication")

	return nil
}
