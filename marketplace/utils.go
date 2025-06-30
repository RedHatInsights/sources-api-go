package marketplace

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/go-redis/redis/v8"
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
	if err != nil {
		// If the error isn't "redis.Nil" something went wrong with Redis.
		if err != redis.Nil {
			return err
		}
		// In this case we can assume the token is not present in redis, so we can request one token to the
		// marketplace, cache it, and assign it to the "extra" field of the authentication.
		apiKey, err := auth.GetPassword()
		if err != nil {
			return err
		}

		// The Api key must be present to be able to send the request to the marketplace
		if auth.Password == nil {
			return errors.New("API key not present for the marketplace authentication")
		}

		marketplaceTokenProvider = GetMarketplaceTokenProvider(*apiKey)

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

	err = auth.SetExtraField("marketplace", token)
	if err != nil {
		return err
	}

	logger.Log.Log(logrus.InfoLevel, "marketplace token included in authentication")

	return nil
}
