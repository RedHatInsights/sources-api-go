package dao

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// setUpBearerToken sets up a fake bearer token to be used in the tests.
func setUpBearerToken() *marketplace.BearerToken {
	expiration := int64(12345)
	token := "tokenTest"

	return &marketplace.BearerToken{
		Expiration: &expiration,
		Token:      &token,
	}
}

// setUpValidMarketplaceAuth sets up a valid authentication which is of the "marketplace" type.
func setUpValidMarketplaceAuth() *m.Authentication {
	return &m.Authentication{AuthType: "marketplace-token", Password: "apiKey"}
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
	auth := setUpValidMarketplaceAuth()
	auth.AuthType = "whatever"

	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}
}

// TestAuthFromVaultMarketplaceCacheHit tests that when there's a cache hit with the marketplace token, it simply
// returns the token on the "extra" field.
func TestAuthFromVaultMarketplaceCacheHit(t *testing.T) {
	// We need to simulate that the Vault is online.
	conf.SecretStore = "vault"

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

	auth := setUpValidMarketplaceAuth()

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	token, ok := auth.Extra["marketplace"].(*marketplace.BearerToken)
	if !ok {
		t.Errorf(`want type "bearer token", got "%s"`, reflect.TypeOf(auth.Extra["marketplace"]))
	}

	expectedToken := setUpBearerToken()

	{
		want := *expectedToken.Token
		got := *token.Token
		if want != got {
			t.Errorf(`invalid token string. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := *expectedToken.Expiration
		got := *token.Expiration
		if want != got {
			t.Errorf(`invalid token expiration. Want "%d", got "%d"`, want, got)
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

	auth := setUpValidMarketplaceAuth()
	auth.Password = "" // set up the empty password to simulate a missing API key

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

	auth := setUpValidMarketplaceAuth()

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

	auth := setUpValidMarketplaceAuth()

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

	auth := setUpValidMarketplaceAuth()

	err := setMarketplaceTokenAuthExtraField(auth)
	if err == nil {
		t.Error("want error, got nil")
	}
}

// TestAuthFromVault tests that when Vault returns a properly formatted authentication, the authFromvault function is
// able to successfully parse it.
func TestAuthFromVault(t *testing.T) {
	// Set up a test authentication.
	now := time.Now()
	lastAvailableCheckedAt := now.Add(time.Duration(-1) * time.Hour)
	createdAt := now.Add(time.Duration(-2) * time.Hour)

	authentication := m.Authentication{
		Name:                    "test-vault-auth",
		AuthType:                "test-authtype",
		Username:                "my-username",
		Password:                "my-password",
		Extra:                   nil,
		AvailabilityStatus:      m.Available,
		LastAvailableAt:         lastAvailableCheckedAt,
		LastCheckedAt:           lastAvailableCheckedAt,
		AvailabilityStatusError: "there was an error, wow",
		ResourceType:            "source",
		ResourceID:              123,
		SourceID:                25,
		CreatedAt:               createdAt,
		Version:                 "500",
	}

	// Use the "ToVaultMap" function to simulate what Vault would store as an authentication.
	vaultData, err := authentication.ToVaultMap()
	if err != nil {
		t.Errorf(`could not transform authentication to Vault map: %s`, err)
	}

	// The authFromVault function expects strings as timestamps, not "time.Time" types. This is a particularity of the
	// tests since the data that comes from Vault will all be strings. In this case though, as we're directly assigning
	// the data to the map, the latter stores it as the types that the "ToVaultMap" function returns, instead of
	// storing the data as strings. This is why we overwrite that data manually.
	data, ok := vaultData["data"].(map[string]interface{})
	if !ok {
		t.Errorf(`wrong type for the secret's data object. Want "map[string]interface{}", got "%s"'`, reflect.TypeOf(vaultData["data"]))
	}
	data["last_available_at"] = authentication.LastAvailableAt.Format(time.RFC3339Nano)

	// We also want to test if the metadata gets correctly unmarshalled.
	version := json.Number(authentication.Version)
	vaultData["metadata"] = map[string]interface{}{
		"created_time": authentication.CreatedAt.Format(time.RFC3339Nano),
		"version":      version,
	}

	// Build the Vault secret.
	vaultSecret := api.Secret{
		Data: vaultData,
	}

	// Call the function under test and check the results.
	resultingAuth := authFromVault(&vaultSecret)

	// We need this if as otherwise the linter complains about possible nil pointer dereferences.
	if resultingAuth == nil {
		t.Errorf(`authFromVault didn't correctly parse the secret. Got a nil authentication`)
	} else {
		{
			want := authentication.Name
			got := resultingAuth.Name
			if want != got {
				t.Errorf(`authentication names are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.AuthType
			got := resultingAuth.AuthType
			if want != got {
				t.Errorf(`authentication types are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.Username
			got := resultingAuth.Username
			if want != got {
				t.Errorf(`authentication usernames are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.Password
			got := resultingAuth.Password
			if want != got {
				t.Errorf(`authentication passwords are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.ResourceType
			got := resultingAuth.ResourceType
			if want != got {
				t.Errorf(`authentication resoource types are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.ResourceID
			got := resultingAuth.ResourceID
			if want != got {
				t.Errorf(`authentication resource IDs are different. Want "%d", got "%d"`, want, got)
			}
		}

		{
			want := authentication.SourceID
			got := resultingAuth.SourceID
			if want != got {
				t.Errorf(`authentication passwords are different. Want "%d", got "%d"`, want, got)
			}
		}

		{
			want := authentication.AvailabilityStatus
			got := resultingAuth.AvailabilityStatus
			if want != got {
				t.Errorf(`authentication availability statuses are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.LastAvailableAt.Format(time.RFC3339Nano)
			got := resultingAuth.LastAvailableAt.Format(time.RFC3339Nano)
			if want != got {
				t.Errorf(`authentication last available at statuses are different. Want "%s", got "%s"`, want, got)
			}
		}

		{
			want := authentication.LastCheckedAt.Format(time.RFC3339Nano)
			got := resultingAuth.LastCheckedAt.Format(time.RFC3339Nano)
			if want != got {
				t.Errorf(`authentication last checked at statuses are different. Want "%s", got "%s"`, want, got)
			}
		}
	}
}

// TestAuthFromDbExtraNoContent tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace. It simulates the extra field not having any
// previous content.
func TestAuthFromDbExtraNoContent(t *testing.T) {
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	conf.SecretStore = "database"

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

	auth := setUpValidMarketplaceAuth()

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	want, err := json.Marshal(setUpBearerToken())
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if !bytes.Equal(want, auth.ExtraDb) {
		t.Errorf(`want "%s", got "%s"`, want, auth.ExtraDb)
	}
}

// TestAuthFromDbExtraPreviousContent tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace. It simulates the extra field having previous
// content.
func TestAuthFromDbExtraPreviousContent(t *testing.T) {
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	conf.SecretStore = "database"

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

	auth := setUpValidMarketplaceAuth()
	auth.ExtraDb = []byte(`{"hello": "world"}`)

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// tmpWant will hold the structure to what we are expecting to get in return when calling the function under test.
	tmpWant := make(map[string]interface{})
	tmpWant["hello"] = "world"
	tmpWant["marketplace"] = setUpBearerToken()

	want, err := json.Marshal(tmpWant)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if !bytes.Equal(want, auth.ExtraDb) {
		t.Errorf(`want "%s", got "%s"`, want, auth.ExtraDb)
	}
}
