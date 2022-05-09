package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
)

// vaultSecretPathFormat defines the format of the path that the secrets will be created with in Vault. The idea is
// to have the following format:
//
// secret/data/<tenant_id>/<resourceType>_<resourceId>_<resourceUuid>
const vaultSecretPathFormat = "secret/data/%d/%s_%d_%s"

// GetAuthenticationDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetAuthenticationDao func(*int64) AuthenticationDao

// getDefaultAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultAuthenticationDao(tenantId *int64) AuthenticationDao {
	if config.IsVaultOn() {
		return &authenticationDaoImpl{
			TenantID: tenantId,
		}
	} else {
		return &authenticationDaoDbImpl{
			TenantID: tenantId,
		}
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetAuthenticationDao = getDefaultAuthenticationDao
}

type authenticationDaoImpl struct {
	TenantID *int64
}

/*
	Listing is kind of tough here - it is basically O(N) where N is the results
	returned from vault. It will get slow probably when there are 100 results to
	fetch. In Vault's documentation they do say fetching is handled in parallel
	so we could potentially fetch multiple at once.

	TODO: Maybe parallelize fetching multiple records with goroutines +
	waitgroup
*/
func (a *authenticationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	// Set start and end index and take into account value of offset and limit
	start := offset
	end := offset + limit

	if end > len(keys) {
		end = len(keys)
	}

	if start >= end {
		return nil, int64(len(keys)), nil
	}

	out := make([]m.Authentication, 0, len(keys))
	for _, val := range keys[start:end] {
		secret, err := a.getKey(val)
		if err != nil {
			return nil, 0, err
		}

		out = append(out, *secret)
	}
	count := int64(len(keys))

	return out, count, nil
}

func (a *authenticationDaoImpl) ListForSource(sourceID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	// Check if sourceID exists
	_, err := GetSourceDao(a.TenantID).GetById(&sourceID)
	if err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

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

func (a *authenticationDaoImpl) ListForApplication(applicationID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	// checking if application exists first
	_, err := GetApplicationDao(a.TenantID).GetById(&applicationID)
	if err != nil {
		return nil, 0, util.NewErrNotFound("application")
	}

	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	auths := make([]m.Authentication, 0)

	for _, key := range keys {
		if strings.HasPrefix(key, fmt.Sprintf("Application_%v", applicationID)) {
			auth, err := a.getKey(key)
			if err != nil {
				return nil, 0, err
			}

			auths = append(auths, *auth)
		}
	}

	return auths, int64(len(auths)), nil
}

func (a *authenticationDaoImpl) ListForApplicationAuthentication(appauthID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
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

func (a *authenticationDaoImpl) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	_, err := GetEndpointDao(a.TenantID).GetById(&endpointID)
	if err != nil {
		return nil, 0, util.NewErrNotFound("endpoint")
	}

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

func (a *authenticationDaoImpl) getAuthsForAppAuth(appAuths []m.ApplicationAuthentication) ([]m.Authentication, error) {
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
func (a *authenticationDaoImpl) GetById(uid string) (*m.Authentication, error) {
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
		return nil, util.NewErrNotFound("authentication")
	}

	return a.getKey(fullKey)
}

func (a *authenticationDaoImpl) Create(auth *m.Authentication) error {
	query := DB.Select("source_id").Where("tenant_id = ?", *a.TenantID)

	switch strings.ToLower(auth.ResourceType) {
	case "application":
		app := m.Application{ID: auth.ResourceID}
		result := query.Model(&app).First(&app)
		if result.Error != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", auth.ResourceType, auth.ResourceID)
		}

		auth.SourceID = app.SourceID
	case "endpoint":
		endpoint := m.Endpoint{ID: auth.ResourceID}
		result := query.Model(&endpoint).First(&endpoint)
		if result.Error != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", auth.ResourceType, auth.ResourceID)
		}

		auth.SourceID = endpoint.SourceID
	case "source":
		auth.SourceID = auth.ResourceID
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}

	auth.ID = uuid.New().String()
	auth.CreatedAt = time.Now()
	path := fmt.Sprintf(vaultSecretPathFormat, *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	out, err := Vault.Write(path, data)
	if err != nil {
		return err
	}

	number, ok := out.Data["version"].(json.Number)
	if !ok {
		return errors.New("failed to cast vault version number to string")
	}
	auth.Version = number.String()

	return nil
}

// Create method _without_ checking if the resource exists. Basically since this
// is the bulk-create method the resource doesn't exist yet and we know the
// source ID is set beforehand.
func (a *authenticationDaoImpl) BulkCreate(auth *m.Authentication) error {
	auth.ID = uuid.New().String()
	auth.CreatedAt = time.Now()
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	out, err := Vault.Write(path, data)
	if err != nil {
		return err
	}
	number, ok := out.Data["version"].(json.Number)
	if !ok {
		return fmt.Errorf("failed to get version number from string")
	}
	auth.Version = number.String()

	return nil
}

func (a *authenticationDaoImpl) Update(auth *m.Authentication) error {
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	out, err := Vault.Write(path, data)
	if err != nil {
		return err
	}
	number, ok := out.Data["version"].(json.Number)
	if !ok {
		return errors.New("failed to cast vault version number to string")
	}
	auth.Version = number.String()

	return nil
}

func (a *authenticationDaoImpl) Delete(uid string) (*m.Authentication, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if strings.HasSuffix(key, uid) {
			path := fmt.Sprintf("secret/metadata/%d/%s", *a.TenantID, key)
			sec, err := Vault.Delete(path)

			return authFromVault(sec), err
		}
	}

	return nil, util.NewErrNotFound("authentication")
}

func (a *authenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

/*
	This method lists all keys for a certain tenant - this is necessary because
	of the fact that we can't search for a key based on name, type etc much like
	a k/v store. (almost like the `vault kv get` and `vault kv put`)
*/
func (a *authenticationDaoImpl) listKeys() ([]string, error) {
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
func (a *authenticationDaoImpl) getKey(path string) (*m.Authentication, error) {
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

	// set tenant id
	auth.TenantID = *a.TenantID
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
	number, ok := metadata["version"].(json.Number)
	if !ok {
		return nil
	}
	auth.Version = number.String()

	if extra != nil {
		auth.Extra = extra
	}

	if data["name"] != nil {
		var name string
		if name, ok = data["name"].(string); !ok {
			return nil
		}
		auth.Name = &name
	}
	if data["authtype"] != nil {
		if auth.AuthType, ok = data["authtype"].(string); !ok {
			return nil
		}
	}
	if data["username"] != nil {
		var username string
		if username, ok = data["username"].(string); !ok {
			return nil
		}
		auth.Username = &username
	}
	if data["password"] != nil {
		password, ok := data["password"].(string)
		if !ok {
			return nil
		}
		auth.Password = util.StringRef(password)
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

	if data["availability_status"] != nil {
		var availabilityStatus string
		if availabilityStatus, ok = data["availability_status"].(string); !ok {
			return nil
		}
		auth.AvailabilityStatus = &availabilityStatus
	}

	if data["last_available_at"] != nil {
		var lastAvailableAt string
		if lastAvailableAt, ok = data["last_available_at"].(string); !ok {
			return nil
		}

		parsedLastAvailableAt, err := time.Parse(time.RFC3339Nano, lastAvailableAt)
		if err != nil {
			return nil
		}
		auth.LastAvailableAt = &parsedLastAvailableAt
	}

	if data["last_checked_at"] != nil {
		var lastCheckedAt string
		if lastCheckedAt, ok = data["last_checked_at"].(string); !ok {
			return nil
		}

		parsedLastCheckedAt, err := time.Parse(time.RFC3339Nano, lastCheckedAt)
		if err != nil {
			return nil
		}
		auth.LastCheckedAt = &parsedLastCheckedAt
	}

	return auth
}

func (a *authenticationDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	a.TenantID = &resource.TenantID
	authentication, err := a.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	return BulkMessageFromSource(&authentication.Source, authentication)
}

func (a *authenticationDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	a.TenantID = &resource.TenantID
	authentication, err := a.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	err = authentication.UpdateBy(updateAttributes)
	if err != nil {
		return nil, err
	}
	err = a.Update(authentication)
	if err != nil {
		return nil, err
	}

	sourceDao := GetSourceDao(a.TenantID)
	source, err := sourceDao.GetById(&authentication.SourceID)
	if err != nil {
		return nil, err
	}
	authentication.Source = *source

	return authentication, nil
}

func (a *authenticationDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	a.TenantID = &resource.TenantID
	auth, err := a.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	auth.TenantID = resource.TenantID
	auth.Tenant = m.Tenant{ExternalTenant: resource.AccountNumber}
	authEvent := auth.ToEvent()
	data, err := json.Marshal(authEvent)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (a *authenticationDaoImpl) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	var err error
	var resourceAuthentications []m.Authentication

	switch authentication.ResourceType {
	case "Source":
		resourceAuthentications, _, err = a.ListForSource(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	case "Endpoint":
		resourceAuthentications, _, err = a.ListForEndpoint(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	case "Application":
		resourceAuthentications, _, err = a.ListForApplication(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	default:
		return nil, fmt.Errorf("unable to fetch authentications for %s", authentication.ResourceType)
	}

	if err != nil {
		return nil, err
	}

	return resourceAuthentications, nil
}

func (a *authenticationDaoImpl) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, err
	}

	foundKeys, err := a.findKeysByResourceTypeAndId(keys, resourceType, resourceIds)
	if err != nil {
		return nil, err
	}

	var authentications = make([]m.Authentication, 0, len(foundKeys))
	for _, key := range foundKeys {
		auth, err := a.getKey(key)
		if err != nil {
			logging.Log.Errorf(`[authentication_id: %s] Authentication could not be fetched: %s`, key, err)
			continue
		}

		authentications = append(authentications, *auth)
	}

	return authentications, nil
}
func (a *authenticationDaoImpl) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	var deletedAuthentications []m.Authentication
	for _, auth := range authentications {
		path := fmt.Sprintf("%s_%d_%s", auth.ResourceType, auth.ResourceID, auth.ID)

		auth, err := a.Delete(path)
		if err != nil {
			logging.Log.Errorf(`[authentication_id: %s] Could not delete authentication: %s`, auth.ID, err)
			continue
		}

		deletedAuthentications = append(deletedAuthentications, *auth)
	}

	return deletedAuthentications, nil
}

// findKeysByResourceTypeAndId returns the list of keys that matched the given resource type and resource ids. An error
// is returned when the regexp used can't be compiled.
func (a *authenticationDaoImpl) findKeysByResourceTypeAndId(keys []string, resourceType string, resourceIds []int64) ([]string, error) {
	var foundKeys []string

	for _, id := range resourceIds {
		// Try to find "<ResourceType>_<ResourceId>_". The final underscore is important since otherwise if we were
		// looking for a string that contains "Source_1" we could also match "Source_11".
		targetRegex := fmt.Sprintf("%s_%d_", resourceType, id)
		regex, err := regexp.Compile(targetRegex)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			if regex.MatchString(key) {
				foundKeys = append(foundKeys, key)
			}
		}
	}

	return foundKeys, nil
}
