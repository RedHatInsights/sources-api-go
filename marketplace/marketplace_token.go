package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	goredis "github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

var redisKeySuffix = "marketplace_token_%d"

// MarketplaceTokenCacher is an struct which implements the "TokenCacher" interface. This helps in abstracting away the
// dependencies and making testing easier.
type MarketplaceTokenCacher struct {
	TenantID int64
}

// FetchToken fetches the token from the Redis cache. It returns the bearer token on a cache hit. It returns
// (nil, redis.NIL) when the token was not found.
func (mtc *MarketplaceTokenCacher) FetchToken() (*BearerToken, error) {
	redisKey := fmt.Sprintf(redisKeySuffix, mtc.TenantID)

	cachedToken, err := redis.Client.Get(context.Background(), redisKey).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, err
		} else {
			logger.Log.Errorf(`[tenant_id: %d] unexpected error when getting the marketplace token from Redis: %s`, mtc.TenantID, err)

			return nil, fmt.Errorf("could not get the marketplace token")
		}
	}

	token := &BearerToken{}
	if err = json.Unmarshal([]byte(cachedToken), &token); err != nil {
		logger.Log.Errorf(`[tenant_id: %d] unexpected error when unmarshalling the marketplace token: %s`, mtc.TenantID, err)

		return nil, errors.New("could not process the marketplace token")
	}

	logger.Log.Log(logrus.InfoLevel, fmt.Sprintf("fetched marketplace token from Redis for tenant %d", mtc.TenantID))

	return token, err
}

// CacheToken sets the token on the Redis cache.
func (mtc *MarketplaceTokenCacher) CacheToken(token *BearerToken) error {
	redisKey := fmt.Sprintf(redisKeySuffix, mtc.TenantID)

	tokenExpiration := time.Unix(*token.Expiration, 0)
	redisExpiration := time.Until(tokenExpiration)

	if redisExpiration <= 0 {
		logger.Log.Errorf(`[tenant_id: %d] an expired marketplace token was tried to be set: %#v`, mtc.TenantID, token)

		return errors.New("the obtained marketplace token has already expired. Try again")
	}

	err := redis.Client.Set(
		context.Background(),
		redisKey,
		token,
		redisExpiration,
	).Err()

	if err != nil {
		logger.Log.Errorf(`[tenant_id: %d] unexpected error when trying to set the marketplace token on Redis: %s`, mtc.TenantID, err)

		return errors.New("could not store the marketplace token")
	}

	logger.Log.Log(logrus.InfoLevel, fmt.Sprintf("marketplace token cached for tenant %d", mtc.TenantID))

	return nil
}
