package dao

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
)

// TestDeleteApplicationAuthentication tests that an applicationAuthentication gets correctly deleted, and its data returned.
func TestDeleteApplicationAuthentication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	deletedApplicationAuthentication, err := applicationAuthenticationDao.Delete(&fixtures.TestApplicationAuthenticationData[0].ID)
	if err != nil {
		t.Errorf("error deleting an applicationAuthentication: %s", err)
	}

	{
		want := fixtures.TestApplicationAuthenticationData[0].ID
		got := deletedApplicationAuthentication.ID

		if want != got {
			t.Errorf(`incorrect applicationAuthentication deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	DropSchema("delete")
}

// TestDeleteApplicationAuthenticationNotExists tests that when an applicationAuthentication that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteApplicationAuthenticationNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	nonExistentId := int64(12345)
	_, err := applicationAuthenticationDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DropSchema("delete")
}

// TestApplicationAuthenticationsByApplicationsDatabase tests that when using a database datastore, the function under
// test only fetches the application authentications related to the given list of applications.
func TestApplicationAuthenticationsByApplicationsDatabase(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("appauthfind")

	tenantId := int64(1)

	var apps []model.Application
	var appAuthsWant []model.ApplicationAuthentication

	for _, appAuth := range fixtures.TestApplicationAuthenticationData {
		if appAuth.TenantID == tenantId {
			apps = append(apps, model.Application{ID: appAuth.ApplicationID})
			appAuthsWant = append(appAuthsWant, model.ApplicationAuthentication{ID: appAuth.ID})
		}
	}

	daoParams := RequestParams{TenantID: &tenantId}
	appAuthDao := GetApplicationAuthenticationDao(&daoParams)
	appAuthsOut, err := appAuthDao.ApplicationAuthenticationsByResource("Source", apps, nil)
	if err != nil {
		t.Error(err)
	}

	if len(appAuthsOut) != len(appAuthsWant) {
		t.Errorf("wrong count of returned app auths, wanted %d, got %d", len(appAuthsWant), len(appAuthsOut))
	}

	// Check the IDs of returned app auths
	for _, aaOut := range appAuthsOut {
		var aaFound bool
		for _, aaWant := range appAuthsWant {
			if aaWant.ID == aaOut.ID {
				aaFound = true
				break
			}
		}
		if !aaFound {
			t.Errorf("application authentication with id = %d returned as output but was not expected", aaOut.ID)
		}
	}

	DropSchema("appauthfind")
}

// TestApplicationAuthenticationsByAuthenticationsDatabase tests that when using a database datastore, the function under
// test only fetches the application authentications related to the given list of authentications.
func TestApplicationAuthenticationsByAuthenticationsDatabase(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("appauthfind")

	// Get all the DAOs we are going to work with.
	authDao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	appAuthDao := GetApplicationAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	// Maximum of authentications to create.
	maxCreatedAuths := 5

	// Store both the authentications and the application authentications for later.
	var createdAuths = make([]model.Authentication, 0, maxCreatedAuths)
	var createdAppAuths = make([]model.ApplicationAuthentication, 0, maxCreatedAuths)
	for i := 0; i < maxCreatedAuths; i++ {
		// Create the authentication.
		auth := setUpValidAuthentication()
		auth.ResourceID = fixtures.TestApplicationData[0].ID
		auth.ResourceType = "Application"

		err := authDao.Create(auth)
		if err != nil {
			t.Errorf(`could not create fixture authentication: %s`, err)
		}

		createdAuths = append(createdAuths, *auth)

		// Create the application authentication.
		appAuth := model.ApplicationAuthentication{
			ApplicationID:    fixtures.TestApplicationData[0].ID,
			AuthenticationID: auth.DbID,
			TenantID:         fixtures.TestTenantData[0].Id,
		}

		err = appAuthDao.Create(&appAuth)
		if err != nil {
			t.Errorf(`could not create fixture application authentication: %s`, err)
		}

		createdAppAuths = append(createdAppAuths, appAuth)
	}

	// Call the function under test.
	dbAppAuths, err := appAuthDao.ApplicationAuthenticationsByResource("NotASource", []model.Application{}, createdAuths)
	if err != nil {
		t.Errorf(`unexpected error when fetching the application authentications: %s`, err)
	}

	// Check that we fetched the correct amount of application authentications.
	{
		want := maxCreatedAuths
		got := len(dbAppAuths)

		if want != got {
			t.Errorf(`incorrect amount of application authentications fetched. Want "%d", got "%d"`, want, got)
		}
	}

	// Check that we fetched the correct application authentications.
	for i := 0; i < maxCreatedAuths; i++ {
		{
			want := createdAppAuths[i].ID
			got := dbAppAuths[i].ID

			if want != got {
				t.Errorf(`incorrect application authentication fetched. Want application authentication with id "%d", got "%d"`, want, got)
			}
		}
		{
			want := createdAuths[i].DbID
			got := dbAppAuths[i].AuthenticationID

			if want != got {
				t.Errorf(`incorrect application authentication fetched. Want application authentication with authentication id "%d", got "%d"`, want, got)
			}
		}
	}

	DropSchema("appauthfind")
}

// TestApplicationAuthenticationListOffsetAndLimit tests that List() in app auth dao returns correct count value
// and correct count of returned objects
func TestApplicationAuthenticationListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("offset_limit")

	appAuthDao := GetApplicationAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	wantCount := int64(len(fixtures.TestApplicationAuthenticationData))

	for _, d := range fixtures.TestDataOffsetLimit {
		appAuths, gotCount, err := appAuthDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the application authentications: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of application authentications, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(appAuths)
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
	DropSchema("offset_limit")
}

func TestApplicationAuthenticationListUserOwnership(t *testing.T) {
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
	appAuthDao := GetApplicationAuthenticationDao(requestParams)

	appAuths, _, err := appAuthDao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	var appAuthsIDs []int64
	for _, appAuth := range appAuths {
		appAuthsIDs = append(appAuthsIDs, appAuth.ID)
	}

	var expectedAppAuthsIDs []int64
	for _, appAuth := range recordsWithUserID.ApplicationAuthentications {
		expectedAppAuthsIDs = append(expectedAppAuthsIDs, appAuth.ID)
	}

	for _, appAuth := range recordsWithoutUserID.ApplicationAuthentications {
		expectedAppAuthsIDs = append(expectedAppAuthsIDs, appAuth.ID)
	}

	if !cmp.Equal(appAuthsIDs, expectedAppAuthsIDs) {
		t.Errorf("Expected application authentication IDS %v are not same with obtained ids: %v", expectedAppAuthsIDs, appAuthsIDs)
	}

	userWithoutOwnRecords, err := CreateUserForUserID(userIDWithoutOwnRecords, user.TenantID)
	if err != nil {
		t.Errorf(`unable to create user: %v`, err)
	}

	requestParams = &RequestParams{TenantID: &user.TenantID, UserID: &userWithoutOwnRecords.Id}
	appAuthDao = GetApplicationAuthenticationDao(requestParams)

	appAuths, _, err = appAuthDao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	appAuthsIDs = []int64{}
	for _, appAuth := range appAuths {
		appAuthsIDs = append(appAuthsIDs, appAuth.ID)
	}

	expectedAppAuthsIDs = []int64{}
	for _, appAuth := range recordsWithoutUserID.ApplicationAuthentications {
		expectedAppAuthsIDs = append(expectedAppAuthsIDs, appAuth.ID)
	}

	if !cmp.Equal(appAuthsIDs, expectedAppAuthsIDs) {
		t.Errorf("Expected application authentication IDS %v are not same with obtained ids: %v", expectedAppAuthsIDs, appAuthsIDs)
	}

	DropSchema(schema)
}
