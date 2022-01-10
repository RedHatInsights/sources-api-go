package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	"github.com/sirupsen/logrus"
)

var redisKeySuffix = "marketplace_token_%d"

type TokenCacher interface {
	// FetchToken fetches a marketplace token from the cache.
	FetchToken() (*marketplace.BearerToken, error)
	// CacheToken sets a marketplace token on the cache.
	CacheToken(token *marketplace.BearerToken) error
}

// MarketplaceTokenCacher is an struct which implements the "TokenCacher" interface. This helps in abstracting away the
// dependencies and making testing easier.
type MarketplaceTokenCacher struct {
	TenantID int64
}

// FetchToken fetches the token from the Redis cache.
func (mtc *MarketplaceTokenCacher) FetchToken() (*marketplace.BearerToken, error) {
	redisKey := fmt.Sprintf(redisKeySuffix, mtc.TenantID)

	cachedToken, err := Client.Get(redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("token not present in Redis: %s", err)
	}

	token := &marketplace.BearerToken{}
	if err = json.Unmarshal([]byte(cachedToken), &token); err != nil {
		return nil, fmt.Errorf("could not unmarshal the cached token: %s", err)
	}

	logger.Log.Log(logrus.InfoLevel, fmt.Sprintf("fetched marketplace token from Redis for tenant %d", mtc.TenantID))

	return token, err
}

// CacheToken sets the token on the Redis cache.
func (mtc *MarketplaceTokenCacher) CacheToken(token *marketplace.BearerToken) error {
	redisKey := fmt.Sprintf(redisKeySuffix, mtc.TenantID)

	tokenExpiration := time.Unix(*token.Expiration, 0)
	redisExpiration := time.Until(tokenExpiration)

	if redisExpiration <= 0 {
		return fmt.Errorf("refusing to cache an expired token")
	}

	err := Client.Set(
		redisKey,
		token,
		redisExpiration,
	).Err()

	if err != nil {
		return fmt.Errorf("could not set marketplace token on redis: %s", token)
	}

	logger.Log.Log(logrus.InfoLevel, fmt.Sprintf("marketplace token cached for tenant %d", mtc.TenantID))

	return nil
}
