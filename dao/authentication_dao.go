package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

type AuthenticationDaoImpl struct {
	TenantID *int64
}

// marketplaceTokenCacher is a variable that holds the "GetMarketplaceTokenCacher" function, or any function that is
// similar to that one. This way we can inject the "TokenCacher" dependency at runtime, which enables easier mocking
// and testing.
var marketplaceTokenCacher redis.TokenCacher

// GetMarketplaceTokenCacher stores a function that returns a TokenCacher
var GetMarketplaceTokenCacher func(*int64) redis.TokenCacher

// GetMarketplaceTokenCacherWithTenantId is the default implementation which returns a default "TokenCacher" instance,
// which is used in the main application.
func GetMarketplaceTokenCacherWithTenantId(tenantId *int64) redis.TokenCacher {
	return &redis.MarketplaceTokenCacher{TenantID: *tenantId}
}

// marketplaceTokenProvider is a variable similar to marketplaceTokenCacher, which holds the function that returns the
// required dependency.
var marketplaceTokenProvider marketplace.TokenProvider

// GetMarketplaceTokenProvider stores a function that retunrs a TokenProvider
var GetMarketplaceTokenProvider func(string) marketplace.TokenProvider

// GetMarketplaceTokenProviderWithApiKey is the default implementation which returns a default "TokenProvider" instance,
// which is used in the main application.
func GetMarketplaceTokenProviderWithApiKey(apiKey string) marketplace.TokenProvider {
	return &marketplace.MarketplaceTokenProvider{ApiKey: &apiKey}
}

/*
	Listing is kind of tough here - it is basically O(N) where N is the results
	returned from vault. It will get slow probably when there are 100 results to
	fetch. In Vault's documentation they do say fetching is handled in parallel
	so we could potentially fetch multiple at once.

	TODO: Maybe parallelize fetching multiple records with goroutines +
	waitgroup
*/
func (a *AuthenticationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	// Handle if the limit is longer than the keys available
	end := 0
	if limit > len(keys) {
		end = len(keys)
	} else {
		end = limit
	}

	// Initialize the marketplace token cacher as it will be used in the underlying "authFromVault" function, inside
	// ".getKey"
	marketplaceTokenCacher = GetMarketplaceTokenCacher(a.TenantID)
	out := make([]m.Authentication, 0, len(keys))
	for _, val := range keys[offset:end] {
		secret, err := a.getKey(val)
		if err != nil {
			return nil, 0, err
		}

		out = append(out, *secret)
	}
	count := int64(len(out))

	return out, count, nil
}

func (a *AuthenticationDaoImpl) ListForSource(sourceID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	out := make([]m.Authentication, 0)

	for _, key := range keys {
		auth, err := a.getKey(key)
		if err != nil {
			return nil, 0, err
		}

		if auth.SourceID == sourceID {
			out = append(out, *auth)
		}
	}

	return out, int64(len(out)), nil
}

func (a *AuthenticationDaoImpl) ListForApplication(applicationID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	app := m.Application{ID: applicationID}
	result := DB.
		Where("tenant_id = ?", *a.TenantID).
		Preload("ApplicationAuthentications").
		First(&app)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	auths, err := a.getAuthsForAppAuth(app.ApplicationAuthentications)
	if err != nil {
		return nil, 0, err
	}

	return auths, int64(len(auths)), nil
}

func (a *AuthenticationDaoImpl) ListForApplicationAuthentication(appauthID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	appauth := m.ApplicationAuthentication{ID: appauthID}
	result := DB.
		Where("tenant_id = ?", *a.TenantID).
		First(&appauth)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	auths, err := a.getAuthsForAppAuth([]m.ApplicationAuthentication{appauth})
	if err != nil {
		return nil, 0, err
	}

	return auths, int64(len(auths)), nil
}

func (a *AuthenticationDaoImpl) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	auths := make([]m.Authentication, 0)

	for _, key := range keys {
		if strings.HasPrefix(key, fmt.Sprintf("Endpoint_%v", endpointID)) {
			auth, err := a.getKey(key)
			if err != nil {
				return nil, 0, err
			}

			auths = append(auths, *auth)
		}
	}

	return auths, int64(len(auths)), nil
}

func (a *AuthenticationDaoImpl) getAuthsForAppAuth(appAuths []m.ApplicationAuthentication) ([]m.Authentication, error) {
	out := make([]m.Authentication, len(appAuths))
	for i, appAuth := range appAuths {
		auth, err := a.getKey(appAuth.VaultPath)
		if err != nil {
			return nil, err
		}

		out[i] = *auth
	}

	return out, nil
}

/*
	Getting by the UID is tough as well - we have to list all the keys and find
	the one with the right suffix before fetching it. So every "show" request
	will always incur 2 reqs to vault. It may be slower but that is a casualty
	of not having an RDMS.
*/
func (a *AuthenticationDaoImpl) GetById(uid string) (*m.Authentication, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, err
	}

	var fullKey string
	for _, key := range keys {
		if strings.HasSuffix(key, uid) {
			fullKey = key
			break
		}
	}

	if fullKey == "" {
		return nil, fmt.Errorf("authentication not found")
	}

	// The token cacher is initialized here because "getKey" has a call to "authFromvault", and it's the only
	// way of getting the tenant id without passing it around.
	marketplaceTokenCacher = GetMarketplaceTokenCacher(a.TenantID)
	return a.getKey(fmt.Sprintf("secret/data/%d/%s", *a.TenantID, fullKey))
}

func (a *AuthenticationDaoImpl) Create(auth *m.Authentication) error {
	query := DB.Select("source_id").Where("tenant_id = ?", *a.TenantID)

	switch auth.ResourceType {
	case "Application":
		app := m.Application{ID: auth.ResourceID}
		result := query.Model(&app).First(&app)
		if result.Error != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", auth.ResourceType, auth.ResourceID)
		}

		auth.SourceID = app.SourceID
	case "Endpoint":
		endpoint := m.Endpoint{ID: auth.ResourceID}
		result := query.Model(&endpoint).First(&endpoint)
		if result.Error != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", auth.ResourceType, auth.ResourceID)
		}

		auth.SourceID = endpoint.SourceID
	case "Source":
		auth.SourceID = auth.ResourceID
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}

	auth.ID = uuid.New().String()
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	out, err := Vault.Write(path, data)
	if err != nil {
		return err
	}
	auth.Version = out.Data["version"].(json.Number).String()

	return nil
}

func (a *AuthenticationDaoImpl) Update(auth *m.Authentication) error {
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	out, err := Vault.Write(path, data)
	if err != nil {
		return err
	}
	auth.Version = out.Data["version"].(json.Number).String()

	return nil
}

func (a *AuthenticationDaoImpl) Delete(uid string) error {
	keys, err := a.listKeys()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if strings.HasSuffix(key, uid) {
			path := fmt.Sprintf("secret/metadata/%d/%s", *a.TenantID, key)
			_, err := Vault.Delete(path)

			return err
		}
	}

	return fmt.Errorf("not found")
}

func (a *AuthenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

/*
	This method lists all keys for a certain tenant - this is necessary because
	of the fact that we can't search for a key based on name, type etc much like
	a k/v store. (almost like the `vault kv get` and `vault kv put`)
*/
func (a *AuthenticationDaoImpl) listKeys() ([]string, error) {
	// List all the keys
	path := fmt.Sprintf("secret/metadata/%d/", *a.TenantID)
	list, err := Vault.List(path)
	if err != nil || list == nil {
		return nil, err
	}

	// data["keys"] is where the objects are returned. it's an array of
	// interfaces but we know they are strings
	var data []interface{}
	var ok bool
	if data, ok = list.Data["keys"].([]interface{}); !ok {
		return nil, fmt.Errorf("bad data came back from vault")
	}

	keys := make([]string, len(data))
	for i, key := range data {
		if keys[i], ok = key.(string); !ok {
			return nil, fmt.Errorf("bad type cast")
		}
	}

	return keys, nil
}

/*
	Fetch a key from Vault (full path, type and id included)
*/
func (a *AuthenticationDaoImpl) getKey(path string) (*m.Authentication, error) {
	secret, err := Vault.Read(fmt.Sprintf("secret/data/%d/%s", *a.TenantID, path))
	if err != nil || secret == nil {
		return nil, fmt.Errorf("authentication not found")
	}

	// parse the secret using our wild and crazy mapping function
	// if it comes back as nil - something went wrong.
	auth := authFromVault(secret)
	if auth == nil {
		return nil, fmt.Errorf("failed to deserialize secret from vault")
	}

	paths := strings.Split(path, "_")
	// the uid is the last part of the path, e.g. Source_2_435-bnsd-4362
	uid := paths[len(paths)-1]
	auth.ID = uid
	return auth, nil
}

/*
	*VERY* important function. This is the function that parses data from Vault
	into an Authentication object. It is basically the inverse of
	Authentication#ToVaultMap().

	If we are to add more fields - they will need to be added here.
*/
func authFromVault(secret *api.Secret) *m.Authentication {
	// first step is to _actually_ extract the data/metadata hashes - which are
	// just map[string]interface{} but the response data type is very generic so
	// we need to infer it ourselves. which is good because we get a lot of type
	// checking this way.
	var data, metadata, extra map[string]interface{}
	var ok bool
	if data, ok = secret.Data["data"].(map[string]interface{}); !ok {
		return nil
	}
	if metadata, ok = secret.Data["metadata"].(map[string]interface{}); !ok {
		return nil
	}

	// time comes back as a Go time.RFC3339Nano which is nice!
	createdAt, err := time.Parse(time.RFC3339Nano, metadata["created_time"].(string))
	if err != nil {
		return nil
	}

	// the `extra` field also comes back as a map just like we stored which is
	// pretty cool. No need to marshal/unmarshal strings!
	if data["extra"] != nil {
		if extra, ok = data["extra"].(map[string]interface{}); !ok {
			return nil
		}
	}

	// Create the authentication object and fill it in - checking types as we
	// go. We explicitly check each type so we can handle it gracefully rather
	// than a panic happening at runtime.
	auth := &m.Authentication{}
	auth.CreatedAt = createdAt
	auth.Version = metadata["version"].(json.Number).String()

	if extra != nil {
		auth.Extra = extra
	}

	if data["name"] != nil {
		if auth.Name, ok = data["name"].(string); !ok {
			return nil
		}
	}
	if data["authtype"] != nil {
		if auth.AuthType, ok = data["authtype"].(string); !ok {
			return nil
		}
	}
	if data["username"] != nil {
		if auth.Username, ok = data["username"].(string); !ok {
			return nil
		}
	}
	if data["password"] != nil {
		if auth.Password, ok = data["password"].(string); !ok {
			return nil
		}
	}
	if data["resource_type"] != nil {
		if auth.ResourceType, ok = data["resource_type"].(string); !ok {
			return nil
		}
	}
	if data["resource_id"] != nil {
		id, err := strconv.ParseInt(data["resource_id"].(string), 10, 64)
		if err != nil {
			return nil
		}
		auth.ResourceID = id
	}
	if data["source_id"] != nil {
		id, err := strconv.ParseInt(data["source_id"].(string), 10, 64)
		if err != nil {
			return nil
		}
		auth.SourceID = id
	}

	// Try to set the marketplace token in the "auth.Extra" field. If the authentication isn't of the "marketplace"
	// type, this whole thing is skipped.
	if err := setMarketplaceTokenAuthExtraField(auth); err != nil {
		logging.Log.Error(err)

		return nil
	}

	return auth
}

func (a *AuthenticationDaoImpl) BulkMessage(_ *int64) (map[string]interface{}, error) {
	bulkMessage := map[string]interface{}{}

	bulkMessage["source"] = m.Source{}
	bulkMessage["endpoints"] = []m.Endpoint{}
	bulkMessage["endpoints"] = []m.Application{}
	bulkMessage["authentications"] = []m.Authentication{}
	bulkMessage["application_authentications"] = []m.ApplicationAuthenticationEvent{}

	return bulkMessage, nil
}

func (a *AuthenticationDaoImpl) FetchAndUpdateBy(_ *int64, _ map[string]interface{}) error {
	/*
		TODO
	*/
	return nil
}

func (a *AuthenticationDaoImpl) ToEventJSON(_ *int64) ([]byte, error) {
	/*
		TODO: we need to obtain uid
		app, err := a.GetById(uid)
		data, _ := json.Marshal(app.ToEvent())
	*/
	return []byte{}, nil
}

// setMarketplaceTokenAuthExtraField tries to put the marketplace token as a JSON string in the "auth.Extra" field
// only if the provided authentication is of the type "marketplace".
func setMarketplaceTokenAuthExtraField(auth *m.Authentication) error {
	// If the authentication isn't a "marketplace" auth, then skip getting the token
	if auth.Name != "marketplace" {
		return nil
	}

	var token *marketplace.BearerToken

	// First try to fetch the token from the cache
	token, err := marketplaceTokenCacher.FetchToken()

	// If it's not present, request one token to the marketplace, cache it, and assign it to the "extra" field
	// of auth
	if err != nil {
		// The Api key must be present to be able to send the request to the marketplace
		if auth.Password == "" {
			return errors.New("API key not present for the marketplace authentication")
		}

		marketplaceTokenProvider = GetMarketplaceTokenProvider(auth.Password)

		token, err = marketplaceTokenProvider.RequestToken()
		if err != nil {
			return fmt.Errorf("could not get token from the marketplace: %s", err)
		}

		// Cache the token. We really don't mind if we cannot properly cache it: we can request another one and
		// return the one that we got. But we log the error for traceability and future debugging.
		err = marketplaceTokenCacher.CacheToken(token)
		if err != nil {
			logging.Log.Errorf("could not cache the token in Redis: %s", err)
		}
	}

	// Serialize the token as a string
	serializedToken, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("could not serialize marketplace token as a JSON string: %s", err)
	}

	if auth.Extra == nil {
		auth.Extra = make(map[string]interface{})
	}

	auth.Extra["marketplace"] = string(serializedToken)
	logging.Log.Log(logrus.InfoLevel, "marketplace token included in authentication")

	return nil
}
