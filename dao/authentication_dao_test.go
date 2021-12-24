package dao

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/hashicorp/vault/api"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	flag.Bool("createdb", false, "create the test database")
	flag.Bool("integration", false, "run unit or integration tests")
	flag.Parse()

	os.Exit(t.Run())
}

// setUpVaultSecret sets up a minimal vault secret that can get parsed.
func setUpVaultSecret() *api.Secret {
	data := make(map[string]interface{})

	data["data"] = make(map[string]interface{})
	data["metadata"] = make(map[string]interface{})
	data["extra"] = make(map[string]interface{})

	metadata, ok := data["metadata"].(map[string]interface{})
	if !ok { // performed to avoid the "forcetypeassert" linter error
		log.Fatal("expected map[string]interface{}, got otherwise")
	}
	metadata["created_time"] = "2021-01-01T00:00:00.999999999Z"
	metadata["version"] = json.Number("versionNumber")

	dataTopLevel, ok := data["data"].(map[string]interface{})
	if !ok { // performed to avoid the "forcetypeassert" linter error
		log.Fatal("expected map[string]interface{}, got otherwise")
	}

	dataTopLevel["name"] = "marketplace"
	dataTopLevel["password"] = "apiKey"

	return &api.Secret{
		Data: data,
	}
}

// setUpBearerToken sets up a fake bearer token to be used in the tests.
func setUpBearerToken() *marketplace.BearerToken {
	expiration := int64(12345)
	token := "tokenTest"

	return &marketplace.BearerToken{
		Expiration: &expiration,
		Token:      &token,
	}
}

// -----------------------------
// --- [Mocks & fakes setup] ---
// -----------------------------
// Here we some mocks and fakes are set up which are required by the following functions:
// - authentication_dao#List
// - authentication_dao#GetById
// - authentication_dao#authFromVault
// By mocking the required dependencies we can easily unit test our logic.

// ---
// Cacher mocks
// ---

// marketplaceTokenCacherSuccessful never returns errors, acts as an always working token cacher.
type marketplaceTokenCacherSuccessful struct {
	TenantID int64
}

func (mtcnc *marketplaceTokenCacherSuccessful) FetchToken() (*marketplace.BearerToken, error) {
	return setUpBearerToken(), nil
}

func (mtcnc *marketplaceTokenCacherSuccessful) CacheToken(token *marketplace.BearerToken) error {
	return nil
}

// marketplaceTokenCacherNotCachedButCacheable acts as if there was no cached token, but the provided token can be
// cacheable.
type marketplaceTokenCacherNotCachedButCacheable struct {
	TenantID int64
}

func (mtcnc *marketplaceTokenCacherNotCachedButCacheable) FetchToken() (*marketplace.BearerToken, error) {
	return nil, fmt.Errorf("token not present")
}

func (mtcnc *marketplaceTokenCacherNotCachedButCacheable) CacheToken(token *marketplace.BearerToken) error {
	return nil
}

// marketplaceTokenCacherNotCached acts as if there was no cached token, and as if trying to cache one raised an error.
type marketplaceTokenCacherNotCached struct {
	TenantID int64
}

func (mtcnc *marketplaceTokenCacherNotCached) FetchToken() (*marketplace.BearerToken, error) {
	return nil, errors.New("token not present")
}

func (mtcnc *marketplaceTokenCacherNotCached) CacheToken(token *marketplace.BearerToken) error {
	return errors.New("unexpected caching error")
}

//---
// Provider mocks
//---
// marketplaceTokenProviderSuccessful acts as if requesting a token to the marketplace returned a proper response.
type marketplaceTokenProviderSuccessful struct {
	ApiKey *string
}

func (marketplaceTokenProviderSuccessful) RequestToken() (*marketplace.BearerToken, error) {
	return setUpBearerToken(), nil
}

// marketplaceTokenProviderFailure acts as if requesting a token returned some kind of error.
type marketplaceTokenProviderFailure struct {
	ApiKey *string
}

func (marketplaceTokenProviderFailure) RequestToken() (*marketplace.BearerToken, error) {
	return nil, errors.New("marketplace is unavailable")
}

// ------------------------------
// --- [/Mocks & fakes setup] ---
// ------------------------------

// TestAuthFromVaultMarketplaceCacheHit tests that when there's a cache hit with the marketplace token, it simply gets
// serialized and properly returned on the "auth.Extra["marketplace]" object.
func TestAuthFromVaultMarketplaceCacheHit(t *testing.T) {
	// For this test we need to simulate that there is a cache hit, so the token cacher must return a proper fake
	// bearer token
	GetMarketplaceTokenCacher = func(tenantId *int64) redis.TokenCacher {
		return &marketplaceTokenCacherSuccessful{
			TenantID: *tenantId,
		}
	}

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	vaultSecret := setUpVaultSecret()
	// Call the function under test
	auth := authFromVault(vaultSecret)
	if auth == nil {
		t.Errorf("want authentication, got nil")
	}

	serializedToken, err := json.Marshal(setUpBearerToken())
	if err != nil {
		t.Errorf("could not serialize the fake token: %s", err)
	}

	if auth != nil { // required to make the SA5011 linter rule "staticcheck" go away
		if auth.Extra == nil {
			t.Errorf(`got nil, want non empty "auth.Extra"`)
		}

		got := auth.Extra["marketplace"]
		want := string(serializedToken)
		if got != want {
			t.Errorf("want %#v, got %#v", want, got)
		}
	}
}

// TestAuthFromVaultMarketplaceProviderEmptyPassword tests that if no API key is stored on Vault, and if there's a
// cache miss, then no token is requested to the marketplace. We cannot ask for a new token if we don't have an API
// key.
func TestAuthFromVaultMarketplaceProviderEmptyPassword(t *testing.T) {
	// In this test we simulate a cache miss, so the token cacher must return an error when requesting a token
	GetMarketplaceTokenCacher = func(tenantId *int64) redis.TokenCacher {
		return &marketplaceTokenCacherNotCached{
			TenantID: *tenantId,
		}
	}

	// As we're simulating the cache miss but a successful marketplace token request, the token provider should return
	// a proper token. Although it will fail because of the empty password bit below.
	GetMarketplaceTokenProvider = func(apiKey string) marketplace.TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	// We are simulating that the Vault secret doesn't have the ApiKey in the password field, so we empty it out
	vaultSecret := setUpVaultSecret()
	dataTopLevel, ok := vaultSecret.Data["data"].(map[string]interface{})
	if !ok { // performed to avoid the "forcetypeassert" linter error
		log.Fatal("map[string]interface{} expected, got otherwise")
	}
	dataTopLevel["password"] = nil

	// Call the function under test
	auth := authFromVault(vaultSecret)
	if auth != nil {
		t.Errorf("want nil auth, got non nil: %v", auth)
	}
}

// TestAuthFromVaultMarketplaceProviderSuccess tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccess(t *testing.T) {
	// In this test we need to simulate a cache miss, and then a proper token caching. So the fake TokenCacher should
	// both miss the cache and be able to cache the provided token.
	GetMarketplaceTokenCacher = func(tenantId *int64) redis.TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// In this test we need the token provider to return a proper token.
	GetMarketplaceTokenProvider = func(apiKey string) marketplace.TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	vaultSecret := setUpVaultSecret()
	// Call the function under test
	auth := authFromVault(vaultSecret)
	if auth == nil {
		t.Error("want non-nil auth, got nil auth")
	}
}

// TestAuthFromVaultMarketplaceProviderSuccessCacheFailure tests that even if caching the requested token from the
// marketplace fails, the process continues and a serialized token is returned. Having an issue when caching the token
// should not impede to return the requested token from the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccessCacheFailure(t *testing.T) {
	// In this test case we need a cache miss but an error when caching the token.
	GetMarketplaceTokenCacher = func(tenantId *int64) redis.TokenCacher {
		return &marketplaceTokenCacherNotCached{
			TenantID: *tenantId,
		}
	}

	// The provider should return a proper token this time.
	GetMarketplaceTokenProvider = func(apiKey string) marketplace.TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	vaultSecret := setUpVaultSecret()
	// Call the function under test
	auth := authFromVault(vaultSecret)
	if auth == nil {
		t.Error("want non-nil auth, got nil auth")
	}
}

// TestAuthFromVaultMarketplaceProviderFailure tests that if there is an error when requesting the token to the
// marketplace, a nil authentication object is returned.
func TestAuthFromVaultMarketplaceProviderFailure(t *testing.T) {
	// In this test the token cacher should simulate a cache miss and the cache function should return no error.
	GetMarketplaceTokenCacher = func(tenantId *int64) redis.TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// The provider in this test case should simulate a failure to request the token to the marketplace.
	GetMarketplaceTokenProvider = func(apiKey string) marketplace.TokenProvider {
		return &marketplaceTokenProviderFailure{
			ApiKey: &apiKey,
		}
	}

	// We need the logging system initialized to not hit a dereference error.
	logging.Log = logrus.New()

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	vaultSecret := setUpVaultSecret()
	// Call the function under test
	auth := authFromVault(vaultSecret)
	if auth != nil {
		t.Errorf("want nil auth, non-nil received: %v", auth)
	}
}
