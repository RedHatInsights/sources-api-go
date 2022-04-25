package dao

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
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
	return &m.Authentication{AuthType: "marketplace-token", Password: util.StringRef("apiKey")}
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
	originalSecretStore := conf.SecretStore
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
	conf.SecretStore = originalSecretStore
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
	auth.Password = nil // set up the empty password to simulate a missing API key

	// Call the function under test
	err := setMarketplaceTokenAuthExtraField(auth)
	if err == nil {
		t.Errorf("want err, got nil")
	}
}

// TestAuthFromVaultMarketplaceProviderSuccess tests that a proper serialized token is returned when there's a cache
// miss and the token is successfully requested to the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccess(t *testing.T) {
	originalSecretStore := conf.SecretStore
	conf.SecretStore = "vault"
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

	conf.SecretStore = originalSecretStore
}

// TestAuthFromVaultMarketplaceProviderSuccessCacheFailure tests that even if caching the requested token from the
// marketplace fails, the process continues and a serialized token is returned. Having an issue when caching the token
// should not impede to return the requested token from the marketplace.
func TestAuthFromVaultMarketplaceProviderSuccessCacheFailure(t *testing.T) {
	originalSecretStore := conf.SecretStore
	conf.SecretStore = "vault"
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
	conf.SecretStore = originalSecretStore
}

// TestAuthFromVaultMarketplaceProviderFailure tests that if there is an error when requesting the token to the
// marketplace, a nil authentication object is returned.
func TestAuthFromVaultMarketplaceProviderFailure(t *testing.T) {
	originalSecretStore := conf.SecretStore
	conf.SecretStore = "vault"
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
	conf.SecretStore = originalSecretStore
}

// TestAuthFromVault tests that when Vault returns a properly formatted authentication, the authFromvault function is
// able to successfully parse it.
func TestAuthFromVault(t *testing.T) {
	// Set up a test authentication.
	now := time.Now()
	lastAvailableCheckedAt := now.Add(time.Duration(-1) * time.Hour)
	createdAt := now.Add(time.Duration(-2) * time.Hour)

	authentication := m.Authentication{
		AuthType:        "test-authtype",
		Extra:           nil,
		LastAvailableAt: &lastAvailableCheckedAt,
		LastCheckedAt:   &lastAvailableCheckedAt,
		ResourceType:    "source",
		ResourceID:      123,
		SourceID:        25,
		CreatedAt:       createdAt,
		Version:         "500",
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
	data["last_checked_at"] = authentication.LastCheckedAt.Format(time.RFC3339Nano)
	// setting the password manually due to the fact that it can be null therefore not in the db. and if it _were_ in
	// the vault db it would come back as a regular string and not a pointer.
	data["password"] = "my-password"
	data["name"] = "test-vault-auth"
	data["username"] = "my-username"
	data["availability_status"] = m.Available
	data["availability_status_error"] = "there was an error, wow"

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
			want := data["name"]
			got := *resultingAuth.Name
			if want != got {
				t.Errorf(`authentication names are different. Want "%v", got "%v"`, want, got)
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
			want := data["username"]
			got := *resultingAuth.Username
			if want != got {
				t.Errorf(`authentication usernames are different. Want "%v", got "%v"`, want, got)
			}
		}

		{
			want := data["password"]
			got := *resultingAuth.Password
			if want != got {
				t.Errorf(`authentication passwords are different. Want "%v", got "%v"`, want, got)
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
			want := data["availability_status"]
			got := *resultingAuth.AvailabilityStatus
			if want != got {
				t.Errorf(`authentication availability statuses are different. Want "%v", got "%v"`, want, got)
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
	originalSecretStore := conf.SecretStore
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	conf.SecretStore = "database"
	// Initialize the encryption key to the following: aaaaaaaaaaaaaaaa
	err := os.Setenv("ENCRYPTION_KEY", "YWFhYWFhYWFhYWFhYWFhYQ")
	if err != nil {
		t.Errorf(`error setting the "ENCRYPTION_KEY" environment variable: %s`, err)
	}
	util.InitializeEncryption()

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
	// Encrypt the password so that the utility can properly decrypt it.
	cypherText, err := util.Encrypt(*auth.Password)
	if err != nil {
		t.Errorf(`could not encrypt the password: %s`, err)
	}
	auth.Password = &cypherText

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err = setMarketplaceTokenAuthExtraField(auth)
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
	conf.SecretStore = originalSecretStore
}

// TestAuthFromDbExtraNullContent is a regression test. It simulates the extra field coming with a valid JSON "null"
// value which resulted in a nil pointer dereference error, since "json.Unmarshal" just unmarshalls that as a "nil"
// value.
func TestAuthFromDbExtraNullContent(t *testing.T) {
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	conf.SecretStore = "database"
	// Initialize the encryption key to the following: aaaaaaaaaaaaaaaa
	err := os.Setenv("ENCRYPTION_KEY", "YWFhYWFhYWFhYWFhYWFhYQ")
	if err != nil {
		t.Errorf(`error setting the "ENCRYPTION_KEY" environment variable: %s`, err)
	}
	util.InitializeEncryption()

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
	// The "null" JSON object is valid and the JSON marshaller simply unmarshals that as "nil" in the target object.
	auth.ExtraDb = datatypes.JSON(`null`)
	cypherText, err := util.Encrypt(*auth.Password)
	if err != nil {
		t.Errorf(`could not encrypt the password: %s`, err)
	}
	auth.Password = &cypherText

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err = setMarketplaceTokenAuthExtraField(auth)
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
	originalSecretStore := conf.SecretStore
	// Simulate that the Vault instance is off, and that we're pulling authentications from the database.
	conf.SecretStore = "database"
	// Initialize the encryption key to the following: aaaaaaaaaaaaaaaa
	err := os.Setenv("ENCRYPTION_KEY", "YWFhYWFhYWFhYWFhYWFhYQ")
	if err != nil {
		t.Errorf(`error setting the "ENCRYPTION_KEY" environment variable: %s`, err)
	}
	util.InitializeEncryption()

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
	// Encrypt the password so that the utility can properly decrypt it.
	cypherText, err := util.Encrypt(*auth.Password)
	if err != nil {
		t.Errorf(`could not encrypt the password: %s`, err)
	}
	auth.Password = &cypherText

	// We need the logging mechanism initialized, as otherwise we will hit a dereference error when trying to use the
	// logger.
	logging.Log = logrus.New()

	// Call the function under test
	err = setMarketplaceTokenAuthExtraField(auth)
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
	conf.SecretStore = originalSecretStore
}

// TestSecretPathDidntChange is a flag test which tells us when the path of the Vault secrets changed. This potentially
// affects "BulkDelete" "keysToMap" and "searchKeys" functions.
func TestSecretPathDidntChange(t *testing.T) {
	tenantId := 5
	resourceType := "Source"
	resourceId := 10
	resourceUuid := "abcd-efgh"

	got := fmt.Sprintf(vaultSecretPathFormat, tenantId, resourceType, resourceId, resourceUuid)

	want := "secret/data/5/Source_10_abcd-efgh"
	if want != got {
		t.Errorf(`the Vault secrets' path changed. Want "%s", got "%s"`, want, got)
	}
}

// TestFindKeysByResourceTypeAndId tests that the function under test returns the expected keys when trying to find
// them by resource type and resource id.
func TestFindKeysByResourceTypeAndId(t *testing.T) {
	testData := []struct {
		// The map of keys we will be receiving as an argument.
		Keys []string
		// The resource type we will tell the search function to search for.
		ResourceType string
		// The resource IDs it will have to try to find.
		ResourceIds []int64
		// The result we expect coming from the function under test.
		ExpectedResult []string
	}{
		{
			Keys: []string{
				"Source_1_uuid",
				"Source_2_uuid",
			},
			ResourceType:   "Source",
			ResourceIds:    []int64{1},
			ExpectedResult: []string{"Source_1_uuid"},
		},
		{
			Keys: []string{
				"Application_1_uuid",
				"Application_2_uuid",
				"Application_31_uuid",
				"Application_255_uuid",
				"Application_412_uuid",
			},
			ResourceType:   "Application",
			ResourceIds:    []int64{1, 100, 255},
			ExpectedResult: []string{"Application_1_uuid", "Application_255_uuid"},
		},
		{
			Keys: []string{
				"Endpoint_31_uuid",
				"Endpoint_412_uuid",
				"Endpoint_500_uuid",
			},
			ResourceType:   "Endpoint",
			ResourceIds:    []int64{1, 31, 500},
			ExpectedResult: []string{"Endpoint_31_uuid", "Endpoint_500_uuid"},
		},
	}

	// We use a RAW impl without the "GetAuthenticationDao" function since we want to access the unexported function.
	implDao := authenticationDaoImpl{TenantID: &fixtures.TestTenantData[0].Id}
	for _, tt := range testData {
		// Call the function under test.
		foundKeys, err := implDao.findKeysByResourceTypeAndId(tt.Keys, tt.ResourceType, tt.ResourceIds)
		if err != nil {
			t.Errorf(`unexpected error when compiling the regular expression for the "findKeysByResourceTypeAndId" function: %s`, err)
		}

		{
			want := len(tt.ExpectedResult)
			got := len(foundKeys)

			if want != got {
				t.Errorf(`the "findKeysByResourceTypeAndId" function found the incorrect amount of keys. Want "%d", got "%d"`, want, got)
			}
		}

		for i := 0; i < len(tt.ExpectedResult); i++ {
			want := tt.ExpectedResult[i]
			got := foundKeys[i]

			if want != got {
				t.Errorf(`the "findKeysByResourceTypeAndId" function found the incorrect key. Want "%s", got "%s"`, want, got)
			}
		}
	}
}

// TestAuthenticationListOffsetAndLimit tests that List() in authentication dao returns correct count value
// and correct count of returned objects
func TestAuthenticationListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")
	originalSecretStore := conf.SecretStore

	wantCount := int64(len(fixtures.TestAuthenticationData))
	Vault = &mocks.MockVault{}

	// Test is running for both options we potentially have => Vault x Database
	// and for each combination of offset and limit in fixtures
	for _, secretStore := range []string{"vault", "database"} {
		conf.SecretStore = secretStore
		authenticationDao := GetAuthenticationDao(&fixtures.TestTenantData[0].Id)

		for _, d := range fixtures.TestDataOffsetLimit {
			fmt.Println(secretStore, d.Limit, d.Offset)
			authentications, gotCount, err := authenticationDao.List(d.Limit, d.Offset, []util.Filter{})
			if err != nil {
				t.Errorf(`unexpected error when listing the authentications: %s`, err)
			}

			if wantCount != gotCount {
				t.Errorf(`incorrect count of authentications, want "%d", got "%d"`, wantCount, gotCount)
			}

			got := len(authentications)
			want := int(wantCount) - d.Offset
			if want < 0 {
				want = 0
			}

			if want > d.Limit {
				want = d.Limit
			}
			if got != want {
				t.Errorf(`objects passed back from DB: want "%v", got "%v"`, want, got)
			}
		}
	}
	DropSchema("offset_limit")
	conf.SecretStore = originalSecretStore
}
