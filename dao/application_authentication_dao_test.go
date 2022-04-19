package dao

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestDeleteApplicationAuthentication tests that an applicationAuthentication gets correctly deleted, and its data returned.
func TestDeleteApplicationAuthentication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestSourceData[0].TenantID)

	applicationAuthentication := fixtures.TestApplicationAuthenticationData[0]
	// Set the ID to 0 to let GORM know it should insert a new applicationAuthentication and not update an existing one.
	applicationAuthentication.ID = 0
	// Set some data to compare the returned applicationAuthentication.
	applicationAuthentication.AuthenticationUID = "complex uuid"

	// Create the test applicationAuthentication.
	err := applicationAuthenticationDao.Create(&applicationAuthentication)
	if err != nil {
		t.Errorf("error creating applicationAuthentication: %s", err)
	}

	deletedApplicationAuthentication, err := applicationAuthenticationDao.Delete(&applicationAuthentication.ID)
	if err != nil {
		t.Errorf("error deleting an applicationAuthentication: %s", err)
	}

	{
		want := applicationAuthentication.ID
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

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestSourceData[0].TenantID)

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

	// Get all the DAOs we are going to work with.
	authDao := GetAuthenticationDao(&fixtures.TestTenantData[0].Id)
	appDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)
	appAuthDao := GetApplicationAuthenticationDao(&fixtures.TestTenantData[0].Id)

	// Maximum of resources to create.
	maxCreatedResources := 5

	// Store the resources for later.
	var createdApps = make([]model.Application, 0, maxCreatedResources)
	var createdAppAuths = make([]model.ApplicationAuthentication, 0, maxCreatedResources)
	for i := 0; i < maxCreatedResources; i++ {
		// Create the authentication.
		auth := setUpValidAuthentication()
		auth.ResourceID = fixtures.TestApplicationData[0].ID
		auth.ResourceType = "Application"

		err := authDao.Create(auth)
		if err != nil {
			t.Errorf(`could not create fixture authentication: %s`, err)
		}

		// Create the application.
		app := model.Application{
			ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
			SourceID:          fixtures.TestSourceData[0].ID,
			TenantID:          fixtures.TestTenantData[0].Id,
		}

		err = appDao.Create(&app)
		if err != nil {
			t.Errorf(`could not create fixture application: %s`, err)
		}

		createdApps = append(createdApps, app)

		// Create the application authentication.
		appAuth := model.ApplicationAuthentication{
			ApplicationID:    app.ID,
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
	dbAppAuths, err := appAuthDao.ApplicationAuthenticationsByResource("Source", createdApps, []model.Authentication{})
	if err != nil {
		t.Errorf(`unexpected error when fetching the application authentications: %s`, err)
	}

	// Check that we fetched the correct amount of application authentications.
	{
		want := maxCreatedResources
		got := len(dbAppAuths)

		if want != got {
			t.Errorf(`incorrect amount of application authentications fetched. Want "%d", got "%d"`, want, got)
		}
	}

	// Check that we fetched the correct application authentications.
	for i := 0; i < maxCreatedResources; i++ {
		{
			want := createdAppAuths[i].ID
			got := dbAppAuths[i].ID

			if want != got {
				t.Errorf(`incorrect application authentication fetched. Want application authentication with id "%d", got "%d"`, want, got)
			}
		}
		{
			want := createdApps[i].ID
			got := dbAppAuths[i].ApplicationID

			if want != got {
				t.Errorf(`incorrect application authentication fetched. Want application authentication with application id "%d", got "%d"`, want, got)
			}
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
	authDao := GetAuthenticationDao(&fixtures.TestTenantData[0].Id)
	appAuthDao := GetApplicationAuthenticationDao(&fixtures.TestTenantData[0].Id)

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
	SwitchSchema("offset_limit")

	appAuthDao := GetApplicationAuthenticationDao(&fixtures.TestTenantData[0].Id)
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
