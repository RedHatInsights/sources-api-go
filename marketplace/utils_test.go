package marketplace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

// setUpBearerToken sets up a fake bearer token to be used in the tests.
func setUpBearerToken() *BearerToken {
	expiration := int64(12345)
	token := "tokenTest"

	return &BearerToken{
		Expiration: &expiration,
		Token:      &token,
	}
}

// setUpValidMarketplaceAuth sets up a valid authentication which is of the "marketplace" type.
func setUpValidMarketplaceAuth() (*m.Authentication, error) {
	// Encrypt the password so that the utility can properly decrypt it.
	cypherText, err := util.Encrypt("apiKey")
	if err != nil {
		return nil, fmt.Errorf(`could not encrypt the password: %s`, err)
	}

	return &m.Authentication{
			AuthType: "marketplace-token",
			Password: &cypherText,
		},
		nil
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

func (mtcnc *marketplaceTokenCacherSuccessful) FetchToken() (*BearerToken, error) {
	return setUpBearerToken(), nil
}

func (mtcnc *marketplaceTokenCacherSuccessful) CacheToken(token *BearerToken) error {
	return nil
}

// marketplaceTokenCacherNotCachedButCacheable acts as if there was no cached token, but the provided token can be
// cacheable.
type marketplaceTokenCacherNotCachedButCacheable struct {
	TenantID int64
}

func (mtcnc *marketplaceTokenCacherNotCachedButCacheable) FetchToken() (*BearerToken, error) {
	return nil, redis.Nil
}

func (mtcnc *marketplaceTokenCacherNotCachedButCacheable) CacheToken(token *BearerToken) error {
	return nil
}

// marketplaceTokenCacherNotCached acts as if there was no cached token, and as if trying to cache one raised an error.
type marketplaceTokenCacherNotCached struct {
	TenantID int64
}

func (mtcnc *marketplaceTokenCacherNotCached) FetchToken() (*BearerToken, error) {
	return nil, redis.Nil
}

func (mtcnc *marketplaceTokenCacherNotCached) CacheToken(token *BearerToken) error {
	return errors.New("unexpected caching error")
}

//---
// Provider mocks
//---
// marketplaceTokenProviderSuccessful acts as if requesting a token to the marketplace returned a proper response.
type marketplaceTokenProviderSuccessful struct {
	ApiKey *string
}

func (marketplaceTokenProviderSuccessful) RequestToken() (*BearerToken, error) {
	return setUpBearerToken(), nil
}

// marketplaceTokenProviderFailure acts as if requesting a token returned some kind of error.
type marketplaceTokenProviderFailure struct {
	ApiKey *string
}

func (marketplaceTokenProviderFailure) RequestToken() (*BearerToken, error) {
	return nil, errors.New("marketplace is unavailable")
}

// ------------------------------
// --- [/Mocks & fakes setup] ---
// ------------------------------

// TestNotMarketplaceAuthNotProcessed tests that in the case of having a non-marketplace authentication, no token is
// fetched whatsoever.
func TestNotMarketplaceAuthNotProcessed(t *testing.T) {
	auth, err := setUpValidMarketplaceAuth()
	auth.AuthType = "whatever"

	if err != nil {
		t.Error(err)
	}

	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}
}

// TestAuthFromVaultMarketplaceCacheHit tests that when there's a cache hit with the marketplace token, it simply
// returns the token on the "extra" field.
func TestAuthFromVaultMarketplaceCacheHit(t *testing.T) {
	// We need to simulate that the Vault is online.
	originalSecretStore := config.Get().SecretStore
	config.Get().SecretStore = "vault"

	// For this test we need to simulate that there is a cache hit, so the token cacher must return a proper fake
	// bearer token
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherSuccessful{
			TenantID: *tenantId,
		}
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	token, ok := auth.Extra["marketplace"].(*BearerToken)
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

	// Restore the original secret store.
	config.Get().SecretStore = originalSecretStore
}

// TestAuthFromVaultMarketplaceProviderEmptyPassword tests that if no API key is stored on Vault, and if there's a
// cache miss, then no token is requested to the marketplace. We cannot ask for a new token if we don't have an API
// key.
func TestAuthFromVaultMarketplaceProviderEmptyPassword(t *testing.T) {
	// In this test we simulate a cache miss, so the token cacher must return an error when requesting a token
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCached{
			TenantID: *tenantId,
		}
	}

	// As we're simulating the cache miss but a successful marketplace token request, the token provider should return
	// a proper token. Although it will fail because of the empty password bit below.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// Set up the empty password to simulate a missing API key.
	auth.Password = nil

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err == nil {
		t.Errorf("want err, got nil")
	}
}

// TestAuthFromVaultMarketplaceProviderSuccess tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccess(t *testing.T) {
	// We need to simulate that the Vault is online.
	originalSecretStore := config.Get().SecretStore
	config.Get().SecretStore = "vault"

	// In this test we need to simulate a cache miss, and then a proper token caching. So the fake TokenCacher should
	// both miss the cache and be able to cache the provided token.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// In this test we need the token provider to return a proper token.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Restore the original secret store.
	config.Get().SecretStore = originalSecretStore
}

// TestAuthFromVaultMarketplaceProviderSuccessCacheFailure tests that even if caching the requested token from the
// marketplace fails, the process continues and a serialized token is returned. Having an issue when caching the token
// should not impede to return the requested token from the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccessCacheFailure(t *testing.T) {
	// We need to simulate that the Vault is online.
	originalSecretStore := config.Get().SecretStore
	config.Get().SecretStore = "vault"

	// In this test case we need a cache miss but an error when caching the token.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCached{
			TenantID: *tenantId,
		}
	}

	// The provider should return a proper token this time.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// Restore the original secret store.
	config.Get().SecretStore = originalSecretStore
}

// TestAuthFromVaultMarketplaceProviderFailure tests that if there is an error when requesting the token to the
// marketplace, a nil authentication object is returned.
func TestAuthFromVaultMarketplaceProviderFailure(t *testing.T) {
	// In this test the token cacher should simulate a cache miss and the cache function should return no error.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// The provider in this test case should simulate a failure to request the token to the marketplace.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderFailure{
			ApiKey: &apiKey,
		}
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err == nil {
		t.Error("want error, got nil")
	}
}

// TestAuthFromDbExtraNoContent tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace. It simulates the extra field not having any
// previous content.
func TestAuthFromDbExtraNoContent(t *testing.T) {
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	config.Get().SecretStore = "database"

	// In this test we need to simulate a cache miss, and then a proper token caching. So the fake TokenCacher should
	// both miss the cache and be able to cache the provided token.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// In this test we need the token provider to return a proper token.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
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

// TestAuthFromDbExtraNullContent is a regression test. It simulates the extra field coming with a valid JSON "null"
// value which resulted in a nil pointer dereference error, since "json.Unmarshal" just unmarshalls that as a "nil"
// value.
func TestAuthFromDbExtraNullContent(t *testing.T) {
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	config.Get().SecretStore = "database"

	// In this test we need to simulate a cache miss, and then a proper token caching. So the fake TokenCacher should
	// both miss the cache and be able to cache the provided token.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// In this test we need the token provider to return a proper token.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	// The "null" JSON object is valid and the JSON marshaller simply unmarshals that as "nil" in the target object.
	auth.ExtraDb = datatypes.JSON(`null`)

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

	// tmpWant will hold the structure to what we are expecting to get in return when calling the function under test.
	tmpWant := make(map[string]interface{})
	tmpWant["marketplace"] = setUpBearerToken()

	want, err := json.Marshal(tmpWant)
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
	config.Get().SecretStore = "database"

	// In this test we need to simulate a cache miss, and then a proper token caching. So the fake TokenCacher should
	// both miss the cache and be able to cache the provided token.
	GetMarketplaceTokenCacher = func(tenantId *int64) TokenCacher {
		return &marketplaceTokenCacherNotCachedButCacheable{
			TenantID: *tenantId,
		}
	}

	// In this test we need the token provider to return a proper token.
	GetMarketplaceTokenProvider = func(apiKey string) TokenProvider {
		return &marketplaceTokenProviderSuccessful{
			ApiKey: &apiKey,
		}
	}

	auth, err := setUpValidMarketplaceAuth()
	if err != nil {
		t.Error(err)
	}

	auth.ExtraDb = []byte(`{"hello": "world"}`)

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	tenantId := int64(5)
	err = SetMarketplaceTokenAuthExtraField(tenantId, auth)
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
