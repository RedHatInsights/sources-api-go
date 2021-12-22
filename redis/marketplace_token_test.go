package redis

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/RedHatInsights/sources-api-go/marketplace"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
)

var tokenCacher = &MarketplaceTokenCacher{TenantID: 5}

// setUpFakeToken sets up a test token ready to be used.
func setUpFakeToken() *marketplace.BearerToken {
	expiration := int64(1609455600) // 2021-01-01T00:00:00
	testApiToken := "testApiToken"

	return &marketplace.BearerToken{
		Expiration: &expiration,
		Token:      &testApiToken,
	}
}

// TestGetTokenBadTenant tests that when given a bad or nonexistent tenant, an expected error is returned.
func TestGetTokenBadTenant(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Errorf("cannot run the Miniredis mock server: %s", err)
	}

	defer mr.Close()

	Client = redis.NewClient(
		&redis.Options{
			Addr: mr.Addr(),
		},
	)

	tokenCacher.TenantID = 12345
	_, err = tokenCacher.FetchToken()
	if err == nil {
		t.Error("want error, got none")
	}
}

// TestGetToken sets up a predefined token on the Redis cache, and tries to fetch it using the "GetToken" function.
func TestGetToken(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Errorf("cannot run the Miniredis mock server: %s", err)
	}

	defer mr.Close()

	Client = redis.NewClient(
		&redis.Options{
			Addr: mr.Addr(),
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

	err = mr.Set(fmt.Sprintf("marketplace_token_%d", tokenCacher.TenantID), string(marshalledToken))
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
	Client = redis.NewClient(&redis.Options{})

	// Set up a fake token and a fake tenant id
	fakeToken := setUpFakeToken()
	tokenCacher.TenantID = 5

	// Call the actual function
	err := tokenCacher.CacheToken(fakeToken)
	if err == nil {
		t.Error("want error, got none")
	}
}

// TestSetTokenSuccess tests that the token is successfully set on Redis.
func TestSetTokenSuccess(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Errorf("cannot run the Miniredis mock server: %s", err)
	}

	defer mr.Close()

	Client = redis.NewClient(
		&redis.Options{
			Addr: mr.Addr(),
		},
	)

	// Set up a fake token and a fake tenant id
	fakeToken := setUpFakeToken()
	tokenCacher.TenantID = 5

	// Call the actual function
	err = tokenCacher.CacheToken(fakeToken)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Fetch the token from Redis
	got, err := mr.Get(fmt.Sprintf("marketplace_token_%d", tokenCacher.TenantID))
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
