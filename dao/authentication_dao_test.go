package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/sirupsen/logrus"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = parser.ParseFlags()

	os.Exit(t.Run())
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

// TestNotMarketplaceAuthNotProcessed tests that in the case of having a non-marketplace authentication, no token is
// fetched whatsoever.
func TestNotMarketplaceAuthNotProcessed(t *testing.T) {
	auth := m.Authentication{Name: "whatever"}

	err := setMarketplaceTokenAuthExtraField(&auth)
	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}
}

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

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	auth := &m.Authentication{Name: "marketplace"}

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
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

	auth := &m.Authentication{Name: "marketplace", Password: ""}

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err == nil {
		t.Errorf("want err, got nil")
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

	auth := &m.Authentication{Name: "marketplace", Password: "12345"}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
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

	auth := &m.Authentication{Name: "marketplace", Password: "12345"}

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
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

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	tenantId := int64(5)
	marketplaceTokenCacher = GetMarketplaceTokenCacher(&tenantId)

	auth := &m.Authentication{Name: "marketplace", Password: "12345"}

	err := setMarketplaceTokenAuthExtraField(auth)
	if err == nil {
		t.Error("want error, got nil")
	}
}
