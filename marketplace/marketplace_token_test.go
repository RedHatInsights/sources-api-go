package marketplace

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	goredis "github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

var tokenCacher = &MarketplaceTokenCacher{TenantID: 5}

// setUpFakeToken sets up a test token ready to be used.
func setUpFakeToken() *BearerToken {
	expiration := time.Now().Add(24 * time.Hour).Unix()
	testApiToken := "testApiToken"

	return &BearerToken{
		Expiration: &expiration,
		Token:      &testApiToken,
	}
}

// TestGetTokenBadTenant tests that when given a bad or nonexistent tenant, an expected error is returned.
func TestGetTokenBadTenant(t *testing.T) {
	redis.Client = goredis.NewClient(
		&goredis.Options{
			Addr: miniredis.Addr(),
		},
	)

	tokenCacher.TenantID = 12345
	_, err := tokenCacher.FetchToken()

	if err != redis.Nil {
		t.Errorf(`want error of type "redis.Nil", got "%s"`, reflect.TypeOf(err))
	}
}

// TestGetToken sets up a predefined token on the Redis cache, and tries to fetch it using the "GetToken" function.
func TestGetToken(t *testing.T) {
	// We need a logger as the cache and uncache functions log what's being done.
	logger.Log = logrus.New()

	redis.Client = goredis.NewClient(
		&goredis.Options{
			Addr: miniredis.Addr(),
		},
	)

	// Set a token on the redis cache, to then try fo fetch it
	token := setUpFakeToken()
	marshalledToken, err := json.Marshal(token)
	if err != nil {
		t.Errorf("no error expected, got %s", err)
	}

	// Use a fake tenant id to set the token on Redis
	tokenCacher.TenantID = 5

	err = miniredis.Set(fmt.Sprintf("marketplace_token_%d", tokenCacher.TenantID), string(marshalledToken))
	if err != nil {
		t.Errorf("no error expected, got %s", err)
	}

	// Fetch the cached token
	cachedToken, err := tokenCacher.FetchToken()
	if err != nil {
		t.Errorf("no error expected, got %s", err)
	}

	// Check that everything matches
	if (*token.Expiration != *cachedToken.Expiration) || (*token.Token != *cachedToken.Token) {
		t.Errorf("want equal tokens, got different ones: [%s] != [%s]", token, cachedToken)
	}
}

// TestSetTokenUnreachableRedis tests that an error is returned when something goes wrong. In this case, an
// unreachable Redis server is simulated.
func TestSetTokenUnreachableRedis(t *testing.T) {
	redis.Client = goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:2345",
		DialTimeout: time.Millisecond,
	})

	// Set up a fake token and a fake tenant id
	fakeToken := setUpFakeToken()
	tokenCacher.TenantID = 5

	// Call the actual function
	err := tokenCacher.CacheToken(fakeToken)

	want := "could not store the marketplace token"
	if want != err.Error() {
		t.Errorf(`unexpected error when caching the token. Want "%s", got "%s"`, want, err)
	}
}

// TestSetTokenSuccess tests that the token is successfully set on Redis.
func TestSetTokenSuccess(t *testing.T) {
	// We need a logger as the cache and uncache functions log what's being done.
	logger.Log = logrus.New()

	redis.Client = goredis.NewClient(
		&goredis.Options{
			Addr: miniredis.Addr(),
		},
	)

	// Set up a fake token and a fake tenant id
	fakeToken := setUpFakeToken()
	tokenCacher.TenantID = 5

	// Call the actual function
	err := tokenCacher.CacheToken(fakeToken)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Fetch the token from Redis
	got, err := miniredis.Get(fmt.Sprintf("marketplace_token_%d", tokenCacher.TenantID))
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Marshal the expected result to compare it with what we received from Redis
	unmarshalledData, err := json.Marshal(fakeToken)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Compare that we fetched the expected token.
	want := string(unmarshalledData)
	if want != got {
		t.Errorf("want %s, got %s", want, got)
	}
}

// TestSetTokenExpired tests that an error is returned when an expired token is trying to be cached.
func TestSetTokenExpired(t *testing.T) {
	// We need a logger as the cache and uncache functions log what's being done.
	logger.Log = logrus.New()

	redis.Client = goredis.NewClient(
		&goredis.Options{
			Addr: miniredis.Addr(),
		},
	)

	// Set up a fake expired token and a fake tenant id
	fakeToken := setUpFakeToken()

	expiredDate := int64(1609455600) // 2021-01-01T00:00:00
	fakeToken.Expiration = &expiredDate

	tokenCacher.TenantID = 5

	// Call the actual function
	err := tokenCacher.CacheToken(fakeToken)

	want := "the obtained marketplace token has already expired. Try again"
	if want != err.Error() {
		t.Errorf("want %s, got %s", want, err)
	}
}
