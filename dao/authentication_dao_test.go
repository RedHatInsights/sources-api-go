package dao

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/vault/api"
)

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
		// Now is test working correctly only for secret store = database
		// fix for vault will be part of next PR and then this condition
		// will be removed
		if secretStore == "vault" {
			break
		}

		conf.SecretStore = secretStore
		authenticationDao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

		for _, d := range fixtures.TestDataOffsetLimit {
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

func TestAuthenticationListUserOwnership(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	schema := "user_ownership"
	SwitchSchema(schema)

	accountNumber := "112567"
	userIDWithOwnRecords := "user_based_user"
	otherUserIDWithOwnRecords := "other_user"
	userIDWithoutOwnRecords := "another_user"

	applicationTypeID := fixtures.TestApplicationTypeData[3].Id
	sourceTypeID := fixtures.TestSourceTypeData[2].Id
	recordsWithUserID, user, err := CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, &userIDWithOwnRecords)
	if err != nil {
		t.Errorf("unable to create source: %v", err)
	}

	_, _, err = CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, &otherUserIDWithOwnRecords)
	if err != nil {
		t.Errorf("unable to create source: %v", err)
	}

	recordsWithoutUserID, _, err := CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, nil)
	if err != nil {
		t.Errorf("unable to create source: %v", err)
	}

	requestParams := &RequestParams{TenantID: &user.TenantID, UserID: &user.Id}
	authenticationDao := GetAuthenticationDao(requestParams)

	authentications, _, err := authenticationDao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	var authenticationIDs []int64
	for _, authentication := range authentications {
		authenticationIDs = append(authenticationIDs, authentication.DbID)
	}

	var expectedAuthenticationIDs []int64
	for _, authentication := range recordsWithUserID.Authentications {
		expectedAuthenticationIDs = append(expectedAuthenticationIDs, authentication.DbID)
	}

	for _, authentication := range recordsWithoutUserID.Authentications {
		expectedAuthenticationIDs = append(expectedAuthenticationIDs, authentication.DbID)
	}

	if !cmp.Equal(authenticationIDs, expectedAuthenticationIDs) {
		t.Errorf("Expected authentication IDs %v are not same with obtained ids: %v", expectedAuthenticationIDs, authenticationIDs)
	}

	userWithoutOwnRecords, err := CreateUserForUserID(userIDWithoutOwnRecords, user.TenantID)
	if err != nil {
		t.Errorf(`unable to create user: %v`, err)
	}

	requestParams = &RequestParams{TenantID: &user.TenantID, UserID: &userWithoutOwnRecords.Id}
	authenticationDao = GetAuthenticationDao(requestParams)

	authentications, _, err = authenticationDao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	authenticationIDs = []int64{}
	for _, authentication := range authentications {
		authenticationIDs = append(authenticationIDs, authentication.DbID)
	}

	expectedAuthenticationIDs = []int64{}
	for _, authentication := range recordsWithoutUserID.Authentications {
		expectedAuthenticationIDs = append(expectedAuthenticationIDs, authentication.DbID)
	}

	if !cmp.Equal(authenticationIDs, expectedAuthenticationIDs) {
		t.Errorf("Expected authentication IDs %v are not same with obtained ids: %v", expectedAuthenticationIDs, authenticationIDs)
	}

	DropSchema(schema)
}
