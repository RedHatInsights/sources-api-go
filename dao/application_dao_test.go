package dao

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// testApplication holds a test application in order to avoid having to write the "fixtures..." stuff every time.
var testApplication = fixtures.TestApplicationData[0]

// TestPausingApplication tests that an application gets correctly paused when using the method from the DAO.
func TestPausingApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("pause_unpause")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)
	err := applicationDao.Pause(testApplication.ID)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	application, err := applicationDao.GetById(&testApplication.ID)
	if err != nil {
		t.Errorf(`error fetching the application. Want nil error, got "%s"`, err)
	}

	want := time.Now()
	if !dateTimesAreSimilar(want, *application.PausedAt) {
		t.Errorf(`want now, got "%s"`, application.PausedAt)
	}

	DropSchema("pause_unpause")
}

// TestResumeApplication tests that the application is properly resumed when using the method from the DAO.
func TestResumeApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("pause_unpause")

	applicationDao := GetApplicationDao(&testApplication.TenantID)
	err := applicationDao.Unpause(testApplication.ID)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	application, err := applicationDao.GetById(&testApplication.ID)
	if err != nil {
		t.Errorf(`error fetching the application. Want nil error, got "%s"`, err)
	}

	var want *time.Time
	if want != application.PausedAt {
		t.Errorf(`want "%s", got "%s"`, want, application.PausedAt)
	}

	DropSchema("pause_unpause")
}

// TestDeleteApplication tests that an application gets correctly deleted, and its data returned.
func TestDeleteApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)

	application := fixtures.TestApplicationData[0]
	// Set the ID to 0 to let GORM know it should insert a new application and not update an existing one.
	application.ID = 0
	// Set some data to compare the returned application.
	application.Extra = []byte(`{"hello": "world"}`)

	// Create the test application.
	err := applicationDao.Create(&application)
	if err != nil {
		t.Errorf("error creating application: %s", err)
	}

	deletedApplication, err := applicationDao.Delete(&application.ID)
	if err != nil {
		t.Errorf("error deleting an application: %s", err)
	}

	{
		want := application.ID
		got := deletedApplication.ID

		if want != got {
			t.Errorf(`incorrect application deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	{
		want := application.Extra
		got := deletedApplication.Extra

		if !bytes.Equal(want, got) {
			t.Errorf(`incorrect application deleted. Want "%s" in the extra field, got "%s"`, want, got)
		}
	}

	DropSchema("delete")
}

// TestDeleteApplicationNotExists tests that when an application that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteApplicationNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := applicationDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DropSchema("delete")
}

func TestApplicationDeleteCascade(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	// Create a new application on the database to cleanly test the function under test.
	applicationDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)
	fixtureApp := m.Application{
		ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
		SourceID:          fixtures.TestSourceData[0].ID,
		TenantID:          fixtures.TestTenantData[0].Id,
	}

	err := applicationDao.Create(&fixtureApp)
	if err != nil {
		t.Errorf(`could not create the fixture application: %s`, err)
	}

	// Create the authentications and the application authentications. The former are needed to avoid the foreign key
	// constraints.
	authenticationDao := GetAuthenticationDao(&fixtures.TestTenantData[0].Id)
	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestTenantData[0].Id)

	// Set the maximum amount of authentications we will create.
	maxAuthenticationsCreated := 5

	// Store the application authentications to perform checks later.
	var createdAppAuths []m.ApplicationAuthentication
	for i := 0; i < maxAuthenticationsCreated; i++ {
		// Create the authentication.
		authentication := setUpValidAuthentication()
		authentication.ResourceType = "Application"
		authentication.ResourceID = fixtureApp.ID

		err := authenticationDao.Create(authentication)
		if err != nil {
			t.Errorf(`could not create the fixture authentication: %s`, err)
		}

		// Create the association between the application and its authentication.
		appAuth := m.ApplicationAuthentication{
			TenantID:          fixtures.TestTenantData[0].Id,
			ApplicationID:     fixtureApp.ID,
			AuthenticationID:  authentication.DbID,
			AuthenticationUID: fmt.Sprintf("%d", i),
		}

		err = applicationAuthenticationDao.Create(&appAuth)
		if err != nil {
			t.Errorf(`could not create the fixture application authentication: %s`, err)
		}

		createdAppAuths = append(createdAppAuths, appAuth)
	}

	deletedApplicationAuthentications, deletedApplication, err := applicationDao.DeleteCascade(fixtureApp.ID)
	if err != nil {
		t.Errorf(`could not delete cascade the application: %s`, err)
	}

	// Count the application authentications from the given application, to check that they were deleted.
	var appAuthCount int64
	err = DB.
		Debug().
		Model(m.ApplicationAuthentication{}).
		Where("application_id = ?", fixtureApp.ID).
		Where("tenant_id = ?", fixtures.TestTenantData[0].Id).
		Count(&appAuthCount).
		Error

	if err != nil {
		t.Errorf(`error counting the application authentications related to the application: %s`, err)
	}

	// Check if the application authentications were deleted or not.
	{
		want := int64(0)
		got := appAuthCount
		if want != got {
			t.Errorf(`the application authentications were not deleted. Want a count of "%d", got "%d"`, want, got)
		}
	}

	// Check that we deleted the correct number of application authentications, and no more.
	{
		want := len(createdAppAuths)
		got := len(deletedApplicationAuthentications)

		if want != got {
			t.Errorf(`unexpected number of application authentications deleted. Want "%d", got "%d"`, want, got)
		}
	}

	// Check that we deleted the application authentications we expected to delete.
	for i := 0; i < maxAuthenticationsCreated; i++ {
		{
			want := createdAppAuths[i].ID
			got := deletedApplicationAuthentications[i].ID

			if want != got {
				t.Errorf(`unexpected application authentication deleted. Want application authentication with ID "%d", got ID "%d"`, want, got)
			}
		}
	}

	// Try to fetch the deleted application.
	var deletedApplicationCheck *m.Application
	err = DB.
		Debug().
		Model(m.Application{}).
		Where(`id = ?`, fixtureApp.ID).
		Where(`tenant_id = ?`, fixtures.TestTenantData[0].Id).
		Find(&deletedApplicationCheck).
		Error

	if err != nil {
		t.Errorf(`unexpected error: %s`, err)
	}

	// Check that the expected application was deleted.
	if deletedApplicationCheck.ID != 0 {
		t.Errorf(`unexpected application fetched. It should be deleted, but this application was fetched: %v`, deletedApplicationCheck)
	}

	{
		want := fixtureApp.ID
		got := deletedApplication.ID

		if want != got {
			t.Errorf(`unexpected application deleted. Want application with id "%d", got "%d"`, want, got)
		}
	}

	// Check that the deleted resources come with the related tenant. This is necessary since otherwise the events will
	// not have the "tenant" key populated.
	for _, applicationAuthentication := range deletedApplicationAuthentications {
		want := fixtures.TestTenantData[0].ExternalTenant
		got := applicationAuthentication.Tenant.ExternalTenant

		if want != got {
			t.Errorf(`the application authentication doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
		}
	}

	want := fixtures.TestTenantData[0].ExternalTenant
	got := deletedApplication.Tenant.ExternalTenant

	if want != got {
		t.Errorf(`the application doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
	}

	DropSchema("delete")
}

// TestApplicationExists tests whether the function exists returns true when the given application exists.
func TestApplicationExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	applicationDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)

	got, err := applicationDao.Exists(fixtures.TestApplicationData[0].ID)
	if err != nil {
		t.Errorf(`unexpected error when checking that the application exists: %s`, err)
	}

	if !got {
		t.Errorf(`the application does exist but the "Exist" function returns otherwise. Want "true", got "%t"`, got)
	}

	DropSchema("exists")
}

// TestApplicationNotExists tests whether the function exists returns false when the given application does not exist.
func TestApplicationNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	applicationDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)

	got, err := applicationDao.Exists(12345)
	if err != nil {
		t.Errorf(`unexpected error when checking that the application exists: %s`, err)
	}

	if got {
		t.Errorf(`the application doesn't exist but the "Exist" function returns otherwise. Want "false", got "%t"`, got)
	}

	DropSchema("exists")
}

// TestApplicationSubcollectionListWithOffsetAndLimit tests that SubCollectionList() in application dao returns
//  correct count value and correct count of returned objects
func TestApplicationSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	applicationDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)
	sourceId := fixtures.TestSourceData[0].ID

	var wantCount int64
	for _, i := range fixtures.TestApplicationData {
		if i.SourceID == sourceId {
			wantCount++
		}
	}

	for _, d := range fixtures.TestDataOffsetLimit {
		applications, gotCount, err := applicationDao.SubCollectionList(m.Source{ID: sourceId}, d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the applications: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of applications, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(applications)
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

// TestApplicationListOffsetAndLimit tests that List() in application dao returns correct count value
// and correct count of returned objects
func TestApplicationListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	applicationDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)
	wantCount := int64(len(fixtures.TestApplicationData))

	for _, d := range fixtures.TestDataOffsetLimit {
		applications, gotCount, err := applicationDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the applications: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of applications, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(applications)
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
