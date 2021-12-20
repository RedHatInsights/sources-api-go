package redis

import (
	"fmt"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	"time"
)

var redisKeySuffix = "marketplace_token_%d"

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
