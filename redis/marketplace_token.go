package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/marketplace"
)

var redisKeySuffix = "marketplace_token_%d"

// GetToken fetches the token from the Redis cache.
func GetToken(tenantId int64) (*marketplace.BearerToken, error) {
	redisKey := fmt.Sprintf(redisKeySuffix, tenantId)

	cachedToken, err := Client.Get(redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("could not fetch the token from Redis: %s", err)
	}

	token := &marketplace.BearerToken{}
	if err = json.Unmarshal([]byte(cachedToken), &token); err != nil {
		return nil, fmt.Errorf("could not unmarshal the cached token: %s", err)
	}

	return token, err
}

// SetToken sets the token on the Redis cache.
func SetToken(tenantId int64, token *marketplace.BearerToken) error {
	redisKey := fmt.Sprintf(redisKeySuffix, tenantId)

	tokenExpiration := time.Unix(*token.Expiration, 0)
	redisExpiration := time.Until(tokenExpiration)

	err := Client.Set(
		redisKey,
		token,
		redisExpiration,
	).Err()

	if err != nil {
		return fmt.Errorf(
			"could not set marketplace token on redis: %s", token,
		)
	}

	return nil
}
