package dao

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
)

// TestAuthType is an authentication type used in the fixtures to perform checks.
const TestAuthType = "test-my-test-auth-type"

// setUpValidAuthentication returns a minimum valid authentication.
func setUpValidAuthentication() *model.Authentication {
	return &model.Authentication{
		AuthType: TestAuthType,
		SourceID: fixtures.TestSourceData[0].ID,
		TenantID: fixtures.TestTenantData[0].Id,
	}
}

// createAuthenticationFixture inserts a new authentication fixture in the database.
func createAuthenticationFixture(t *testing.T) {
	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	auth := setUpValidAuthentication()

	err := dao.BulkCreate(auth)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}
}

// TestAuthenticationDbCreate tests that the "create" function does not present any problems at creating new entities
// if the minimum data is provided for an authentication.
func TestAuthenticationDbCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	auth := setUpValidAuthentication()

	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(auth)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbBulkCreate tests that the "BulkCreate" function does not present any problems at creating new
// entities if the minimum data is provided for an authentication.
func TestAuthenticationDbBulkCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	auth := setUpValidAuthentication()

	err := dao.BulkCreate(auth)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbList tests that the "list" operation returns the expected number of authentications.
func TestAuthenticationDbList(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	// Create another authentication to see if the listing function also brings it back.
	createAuthenticationFixture(t)

	authentications, count, err := dao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`could not fetch the authentications from the database: %s`, err)
	}

	// We should have the authentication from the fixtures plus the one we just created.
	want := len(fixtures.TestAuthenticationData) + 1
	got := int(count)
	if want != got {
		t.Errorf(`incorrect number of authentications fetched. Want "%d", got "%d"`, want, got)
	}

	// Check that we can find the inserted fixture in the list.
	var foundInsertedFixture bool
	for _, auth := range authentications {
		if auth.AuthType == TestAuthType {
			foundInsertedFixture = true
		}
	}

	if !foundInsertedFixture {
		t.Errorf(`the fixture that was inserted did not come in the authentications list. Something went wrong.`)
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbGet tests that the "get" operation is able to fetch the expected authentication.
func TestAuthenticationDbGetById(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	id := strconv.FormatInt(authFixture.DbID, 10)

	dbAuth, err := dao.GetById(id)
	if err != nil {
		t.Errorf(`could not fetch the authentication from the database: %s`, err)
	}

	want := TestAuthType
	got := dbAuth.AuthType
	if want != got {
		t.Errorf(`wrong authentication fetched. Want "%s" authtype, got "%s"`, want, got)
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbUpdate tests that the "update" operation is able to properly update the authentication.
func TestAuthenticationDbUpdate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	// Update the authentication type.
	updatedAuthType := "new-fresh-authtype"
	authFixture.AuthType = updatedAuthType

	err = dao.Update(authFixture)
	if err != nil {
		t.Errorf(`error updating the authentication: %s`, err)
	}

	id := strconv.FormatInt(authFixture.DbID, 10)

	// Fetch the updated authentication.
	updatedAuthentication, err := dao.GetById(id)
	if err != nil {
		t.Errorf(`error fetching the authentication: %s`, err)
	}

	want := updatedAuthType
	got := updatedAuthentication.AuthType
	if want != got {
		t.Errorf(`deleted the wrong authentication. Want authtype "%s", got "%s"`, want, got)
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbGet tests that the "delete" operation is able to delete the expected authentication.
func TestAuthenticationDbDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	id := strconv.FormatInt(authFixture.DbID, 10)

	deletedAuth, err := dao.Delete(id)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	{
		want := TestAuthType
		got := deletedAuth.AuthType
		if want != got {
			t.Errorf(`deleted the wrong authentication. Want authtype "%s", got "%s"`, want, got)
		}
	}

	_, err = dao.GetById(id)
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestAuthenticationDbGet tests the "delete" operation returns a "not found" error when trying to delete a
// non-existing authentication.
func TestAuthenticationDbDeleteNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	_, err := dao.Delete("12345")
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestTenantId is a trivial test which tests that a correct tenant ID is returned in the function.
func TestTenantId(t *testing.T) {
	tenantId := int64(12345)
	dao := GetAuthenticationDao(&RequestParams{TenantID: &tenantId})

	want := tenantId
	got := dao.Tenant()

	if want != *got {
		t.Errorf(`incorrect tenant ID returned. Want "%d", got "%d"`, want, got)
	}
}

// TestListForSource tests if "list for source" only lists the authentications of the related source, and no more.
func TestListForSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create a new source the new fixtures will be attached to.
	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	uuidRaw, _ := uuid.NewUUID()
	uuidStr := uuidRaw.String()
	source := model.Source{
		Name:         "new source in new tenant",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     fixtures.TestTenantData[1].Id,
		Uid:          &uuidStr,
	}

	err := sourceDao.Create(&source)
	if err != nil {
		t.Errorf(`error creating a source: %s`, err)
	}

	// Create three new authentications for the new source.
	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	var i int
	var maxAuths = 3
	for i < maxAuths {
		auth := &model.Authentication{
			AuthType:     TestAuthType,
			ResourceID:   source.ID,
			ResourceType: "Source",
			TenantID:     fixtures.TestTenantData[1].Id,
			SourceID:     source.ID,
		}

		// Using bulk create so we don't check to see if the resource is there first
		if err = dao.BulkCreate(auth); err != nil {
			t.Errorf(`error creating authentication: %s`, err)
		}

		i++
	}

	// Call the function under test.
	authentications, count, err := dao.ListForSource(source.ID, 100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`[source_id: %d] error listing the authentications for the source: %s`, source.ID, err)
	}

	want := maxAuths
	got := int(count)
	if want != got {
		t.Errorf(`incorrect amount of authentications fetched. Want "%d", got "%d"`, want, got)
	}

	for _, auth := range authentications {
		if auth.AuthType != TestAuthType {
			t.Errorf(`incorrect authentication fetched. Want authtype "%s", got "%s"`, TestAuthType, auth.AuthType)
		}
	}

	DropSchema("authentications_db")
}

// TestListForSourceNotFound tests if a not found error is returned when a nonexistent source is given to the function
// under test.
func TestListForSourceNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})

	// Call the function under test.
	_, _, err := dao.ListForSource(12345, 100, 0, []util.Filter{})

	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestListForApplication tests if "list for Application" only lists the authentications of the related application, and no
// more.
func TestListForApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create a new source the new fixtures will be attached to.
	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	uuidRaw, _ := uuid.NewUUID()
	uuidStr := uuidRaw.String()
	source := model.Source{
		Name:         "new source in new tenant",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     fixtures.TestTenantData[1].Id,
		Uid:          &uuidStr,
	}

	err := sourceDao.Create(&source)
	if err != nil {
		t.Errorf(`error creating a source: %s`, err)
	}

	// Create an application fixture.
	applicationDao := GetApplicationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	application := model.Application{
		ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
		SourceID:          source.ID,
	}

	err = applicationDao.Create(&application)
	if err != nil {
		t.Errorf(`error creating an application: %s`, err)
	}

	// Create three new authentications for the new application.
	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	var i int
	var maxAuths = 3
	for i < maxAuths {
		auth := &model.Authentication{
			AuthType:     TestAuthType,
			ResourceID:   application.ID,
			ResourceType: "Application",
			TenantID:     fixtures.TestTenantData[1].Id,
		}

		// Using bulk create so we don't check to see if the resource is there first
		if err = dao.BulkCreate(auth); err != nil {
			t.Errorf(`error creating authentication: %s`, err)
		}

		i++
	}

	// Call the function under test.
	authentications, count, err := dao.ListForApplication(application.ID, 100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`[application_id: %d] error listing the authentications for the application: %s`, application.ID, err)
	}

	want := maxAuths
	got := int(count)
	if want != got {
		t.Errorf(`incorrect amount of authentications fetched. Want "%d", got "%d"`, want, got)
	}

	for _, auth := range authentications {
		if auth.AuthType != TestAuthType {
			t.Errorf(`incorrect authentication fetched. Want authtype "%s", got "%s"`, TestAuthType, auth.AuthType)
		}
	}

	DropSchema("authentications_db")
}

// TestListForApplicationNotFound tests if a not found error is returned when a nonexistent application is given to the
// function under test.
func TestListForApplicationNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})

	// Call the function under test.
	_, _, err := dao.ListForApplication(12345, 100, 0, []util.Filter{})

	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestListForApplicationAuthentication tests if "list for ApplicationAuthentication" only lists the authentications of
// the related ApplicationAuthentication, and no more.
func TestListForApplicationAuthentication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create a new source the new fixtures will be attached to.
	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	uuidRaw, _ := uuid.NewUUID()
	uuidStr := uuidRaw.String()
	source := model.Source{
		Name:         "new source in new tenant",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     fixtures.TestTenantData[1].Id,
		Uid:          &uuidStr,
	}

	err := sourceDao.Create(&source)
	if err != nil {
		t.Errorf(`error creating a source: %s`, err)
	}

	// Create an application fixture.
	applicationDao := GetApplicationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	application := model.Application{
		ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
		SourceID:          source.ID,
	}

	err = applicationDao.Create(&application)
	if err != nil {
		t.Errorf(`error creating an application: %s`, err)
	}

	// Create a new authentication for the new application authentication.
	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	auth := &model.Authentication{
		AuthType:     TestAuthType,
		ResourceType: "Application",
		ResourceID:   application.ID,
		SourceID:     source.ID,
		TenantID:     fixtures.TestTenantData[1].Id,
	}

	// Using bulk create so we don't check to see if the resource is there first
	if err = dao.BulkCreate(auth); err != nil {
		t.Errorf(`error creating authentication: %s`, err)
	}

	// Create the application authentication.
	appAuthDao := GetApplicationAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	appAuth := model.ApplicationAuthentication{
		TenantID:         fixtures.TestTenantData[1].Id,
		ApplicationID:    application.ID,
		AuthenticationID: auth.DbID,
	}

	err = appAuthDao.Create(&appAuth)
	if err != nil {
		t.Errorf(`error creating application authentication: %s`, err)
	}

	// Call the function under test.
	authentications, count, err := dao.ListForApplicationAuthentication(appAuth.ID, 100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`[application_authentication_id: %d] error listing the authentications for the application authentication: %s`, appAuth.ID, err)
	}

	want := 1
	got := int(count)
	if want != got {
		t.Errorf(`incorrect amount of authentications fetched. Want "%d", got "%d"`, want, got)
	}

	for _, auth := range authentications {
		if auth.AuthType != TestAuthType {
			t.Errorf(`incorrect authentication fetched. Want authtype "%s", got "%s"`, TestAuthType, auth.AuthType)
		}
	}

	DropSchema("authentications_db")
}

// TestListForApplicationAuthenticationNotFound tests if a not found error is returned when a nonexistent application
// authentication is given to the function under test.
func TestListForApplicationAuthenticationNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})

	// Call the function under test.
	_, _, err := dao.ListForApplicationAuthentication(12345, 100, 0, []util.Filter{})

	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestListForEndpoint tests if "list for Endpoint" only lists the authentications of the related endpoint, and no more.
func TestListForEndpoint(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create a new source the new fixtures will be attached to.
	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	uuidRaw, _ := uuid.NewUUID()
	uuidStr := uuidRaw.String()
	source := model.Source{
		Name:         "new source in new tenant",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     fixtures.TestTenantData[1].Id,
		Uid:          &uuidStr,
	}

	err := sourceDao.Create(&source)
	if err != nil {
		t.Errorf(`error creating a source: %s`, err)
	}

	// Create an endpoint fixture.
	endpointDao := GetEndpointDao(&fixtures.TestTenantData[1].Id)
	endpoint := model.Endpoint{
		SourceID: source.ID,
	}

	err = endpointDao.Create(&endpoint)
	if err != nil {
		t.Errorf(`error creating an endpoint: %s`, err)
	}

	// Create three new authentications for the new application authentication.
	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})
	var i int
	var maxAuths = 3
	for i < maxAuths {
		auth := &model.Authentication{
			AuthType:     TestAuthType,
			ResourceID:   endpoint.ID,
			ResourceType: "Endpoint",
			TenantID:     fixtures.TestTenantData[1].Id,
		}

		// Using bulk create so we don't check to see if the resource is there first
		if err = dao.BulkCreate(auth); err != nil {
			t.Errorf(`error creating authentication: %s`, err)
		}

		i++
	}

	// Call the function under test.
	authentications, count, err := dao.ListForEndpoint(endpoint.ID, 100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`[endpoint_id: %d] error listing the authentications for the endpoint authentication: %s`, endpoint.ID, err)
	}

	want := maxAuths
	got := int(count)
	if want != got {
		t.Errorf(`incorrect amount of authentications fetched. Want "%d", got "%d"`, want, got)
	}

	for _, auth := range authentications {
		if auth.AuthType != TestAuthType {
			t.Errorf(`incorrect authentication fetched. Want authtype "%s", got "%s"`, TestAuthType, auth.AuthType)
		}
	}

	DropSchema("authentications_db")
}

// TestListForEndpointNotFound tests if a not found error is returned when a nonexistent endpoint is given to the
// function under test.
func TestListForEndpointNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[1].Id})

	// Call the function under test.
	_, _, err := dao.ListForEndpoint(12345, 0, 0, []util.Filter{})

	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DropSchema("authentications_db")
}

// TestFetchAndUpdateBy tests if "FetchAndUpdateBy" updates the timestamps as expected.
func TestFetchAndUpdateBy(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	// Convert the ID to a string format to be able to update/fetch it.
	id := strconv.FormatInt(authFixture.DbID, 10)

	// Create the attributes that we will be updating in the authentication.
	now := time.Now()
	availabilityStatus := "inventedStatus"
	availabilityStatusError := "inventedError"

	attributes := map[string]interface{}{
		"last_checked_at":           now.Format(time.RFC3339Nano),
		"last_available_at":         now.Format(time.RFC3339Nano),
		"availability_status":       availabilityStatus,
		"availability_status_error": availabilityStatusError,
	}

	resource := util.Resource{
		ResourceUID: id,
		TenantID:    fixtures.TestTenantData[0].Id,
	}

	// Call the function under test.
	_, err = dao.FetchAndUpdateBy(resource, attributes)
	if err != nil {
		t.Errorf(`error with "FetchAndUpdateBy" function: %s`, err)
	}

	// Fetch the authentication and check if it was correctly updated.
	dbAuth, err := dao.GetById(id)
	if err != nil {
		t.Errorf(`could not fetch the authentication from the database: %s`, err)
	}

	{
		want := now
		got := *dbAuth.LastCheckedAt

		if !dateTimesAreSimilar(want, got) {
			t.Errorf(`authentication was not updated. Want "last checked at" "%s", got "%s"`, want, got)
		}
	}

	{
		want := now
		got := *dbAuth.LastAvailableAt

		if !dateTimesAreSimilar(want, got) {
			t.Errorf(`authentication was not updated. Want "last available at" "%s", got "%s"`, want, got)
		}
	}

	{
		want := availabilityStatus
		got := *dbAuth.AvailabilityStatus

		if want != got {
			t.Errorf(`authentication was not updated. Want "availability status" "%s", got "%s"`, want, got)
		}
	}

	{
		want := availabilityStatusError
		got := *dbAuth.AvailabilityStatusError

		if want != got {
			t.Errorf(`authentication was not updated. Want "availability status error" "%s", got "%s"`, want, got)
		}
	}

	DropSchema("authentications_db")
}

// TestToEventJSON tests if "FetchAndUpdateBy" returns the expected output for the given resource.
func TestToEventJSON(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	// Using bulk create so we don't check to see if the resource is there first
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	// Convert the ID to a string format to be able to fetch it.
	id := strconv.FormatInt(authFixture.DbID, 10)

	// Fetch the authentication and "convert it to an event", to then check the output from the function under test.
	dbAuth, err := dao.GetById(id)
	if err != nil {
		t.Errorf(`could not fetch the authentication from the database: %s`, err)
	}
	want, err := json.Marshal(dbAuth.ToEvent())
	if err != nil {
		t.Errorf(`error marshalling the authentication fixture to JSON: %s`, err)
	}

	// Call the function under test.
	resource := util.Resource{
		ResourceUID: id,
		TenantID:    fixtures.TestTenantData[0].Id,
	}
	got, err := dao.ToEventJSON(resource)
	if err != nil {
		t.Errorf(`error on "ToEventJSON": %s`, err)
	}

	if !bytes.Equal(want, got) {
		t.Errorf(`"ToEventJSON" didn't return the expected result. Want "%s", got "%s"'`, want, got)
	}

	DropSchema("authentications_db")
}

// TestBulkMessage tests if "BulkMessage" returns the expected output for the given resource. It simply calls the
// function under test and "BulkMessageFromSource", and then compares outputs, since it's not the aim of this test to
// test also "BulkMessageFromSource".
func TestBulkMessage(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Create the authentication fixture that we will be fetching.
	authFixture := setUpValidAuthentication()
	authFixture.ResourceID = fixtures.TestSourceData[0].ID
	authFixture.ResourceType = "Source"

	dao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	err := dao.BulkCreate(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	// Convert the ID to a string format to be able to fetch it.
	id := strconv.FormatInt(authFixture.DbID, 10)

	// Fetch the authentication to be able to use it in the "BulkMessageFromSource" function.
	dbAuth, err := dao.GetById(id)
	if err != nil {
		t.Errorf(`could not fetch the authentication from the database: %s`, err)
	}
	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	source, err := sourceDao.GetById(&authFixture.SourceID)
	if err != nil {
		t.Errorf(`could not fetch source: %s`, err)
	}

	// Call the service function and store the expected output to compare it later.
	serviceOutput, err := BulkMessageFromSource(source, dbAuth)
	if err != nil {
		t.Errorf(`unexpected error in "BulkMessageFromSource": %s`, err)
	}

	// Call the function under test.
	resource := util.Resource{
		ResourceUID: id,
		TenantID:    fixtures.TestTenantData[0].Id,
	}
	bulkMessageOutput, err := dao.BulkMessage(resource)
	if err != nil {
		t.Errorf(`error on "ToEventJSON": %s`, err)
	}

	want, err := json.Marshal(serviceOutput)
	if err != nil {
		t.Errorf(`unexpected error when generating the expected output from the service function: %s`, err)
	}

	got, err := json.Marshal(bulkMessageOutput)
	if err != nil {
		t.Errorf(`unexpected error when generating the result output from the "BulkMessage" function: %s`, err)
	}

	if !bytes.Equal(want, got) {
		t.Errorf(`"BulkMessage" didn't return the expected result. Want "%s", got "%s"'`, want, got)
	}

	DropSchema("authentications_db")
}

// TestListIdsForResource tests that the function under test is able to fetch the authentications belonging to a
// resource type and a resource id.
func TestListIdsForResource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// In this test we want a clean "authentications" table.
	err := DB.
		Debug().
		Model(model.Authentication{}).
		Delete(fixtures.TestAuthenticationData).
		Error
	if err != nil {
		t.Errorf(`could not delete the previously inserted fixtures`)
	}

	// Specify the resources we will be creating the authentications for.
	resources := []struct {
		ResourceType string
		// We need a valid resource ID so that we are able to create a valid authentication, since there is a foreign
		// key which prevents creating authentications with random resource ids.
		ValidResourceId int64
	}{
		{
			ResourceType:    "Source",
			ValidResourceId: fixtures.TestSourceData[0].ID,
		},
		{
			ResourceType:    "Application",
			ValidResourceId: fixtures.TestApplicationData[0].ID,
		},
		{
			ResourceType:    "Endpoint",
			ValidResourceId: fixtures.TestEndpointData[0].ID,
		},
	}

	// How many authentications will we be creating per resource?
	maxAuthenticationsPerResource := 5

	authsDao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	// Create the authentications.
	for _, resource := range resources {
		var createdAuthentications = make([]model.Authentication, 0, maxAuthenticationsPerResource)
		var resourceIds = make([]int64, 0, maxAuthenticationsPerResource)
		for i := 0; i < maxAuthenticationsPerResource; i++ {
			authFixture := setUpValidAuthentication()
			authFixture.ResourceID = resource.ValidResourceId
			authFixture.ResourceType = resource.ResourceType
			// The fake username is used to make sure we are fetching the right authentication.
			fakeUsername := fmt.Sprintf("%s-%d", resource.ResourceType, i)
			authFixture.Username = &fakeUsername

			err := authsDao.Create(authFixture)
			if err != nil {
				t.Errorf(`error creating the authentication: %s`, err)
			}

			resourceIds = append(resourceIds, authFixture.ResourceID)
			createdAuthentications = append(createdAuthentications, *authFixture)
		}

		// Fetch the freshly created authentications.
		fetchedAuthentications, err := authsDao.ListIdsForResource(resource.ResourceType, resourceIds)
		if err != nil {
			t.Errorf(`[resource_type: %s][resource_ids: %v] Error fetching the authentications: %s`, resource.ResourceType, resourceIds, err)
		}

		// We shouldn't have fetched more authentications than the ones we have inserted.
		{
			want := len(createdAuthentications)
			got := len(fetchedAuthentications)
			if want != got {
				t.Errorf(`incorrect number of authentications fetched. Want "%d", got "%d"`, want, got)
			}
		}

		// Perform some more checks to be sure that we fetched the correct authentications.
		for i := 0; i < len(createdAuthentications); i++ {
			{
				want := createdAuthentications[i].DbID
				got := fetchedAuthentications[i].DbID

				if want != got {
					t.Errorf(`unexpected authentication fetched. Want authentication with id "%d", got "%d"`, want, got)
				}
			}
			{
				want := createdAuthentications[i].ResourceID
				got := fetchedAuthentications[i].ResourceID

				if want != got {
					t.Errorf(`unexpected authentication fetched. Want authentication with resource id "%d", got "%d"`, want, got)
				}
			}
			{
				want := createdAuthentications[i].ResourceType
				got := fetchedAuthentications[i].ResourceType

				if want != got {
					t.Errorf(`unexpected authentication fetched. Want authentication with resource type "%s", got "%s"`, want, got)
				}
			}
			{
				want := *createdAuthentications[i].Username
				got := *fetchedAuthentications[i].Username

				if want != got {
					t.Errorf(`unexpected authentication fetched. Want authentication with username "%s", got "%s"`, want, got)
				}
			}
		}
	}

	DropSchema("authentications_db")
}

// TestBulkDelete tests that the BulkDelete function only deletes the authentications that were passed. N
// authentications are created for multiple resource types and the function tries to bulk delete them all.
func TestBulkDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// Specify the resources we will be creating the authentications for.
	resources := []struct {
		ResourceType string
		// We need a valid resource ID so that we are able to create a valid authentication, since there is a foreign
		// key which prevents creating authentications with random resource ids.
		ValidResourceId int64
	}{
		{
			ResourceType:    "Source",
			ValidResourceId: fixtures.TestSourceData[0].ID,
		},
		{
			ResourceType:    "Application",
			ValidResourceId: fixtures.TestApplicationData[0].ID,
		},
		{
			ResourceType:    "Endpoint",
			ValidResourceId: fixtures.TestEndpointData[0].ID,
		},
	}

	// How many authentications will we be creating per resource?
	maxAuthenticationsPerResource := 5

	authsDao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	// Store the authentications for later.
	var createdAuthentications = make([]model.Authentication, 0, len(resources)*maxAuthenticationsPerResource)

	// Create the authentications.
	for _, resource := range resources {
		for i := 0; i < maxAuthenticationsPerResource; i++ {
			authFixture := setUpValidAuthentication()
			authFixture.ResourceID = resource.ValidResourceId
			authFixture.ResourceType = resource.ResourceType
			// The fake username is used to make sure we are fetching the right authentication.
			fakeUsername := fmt.Sprintf("%s-%d", resource.ResourceType, i)
			authFixture.Username = &fakeUsername

			err := authsDao.Create(authFixture)
			if err != nil {
				t.Errorf(`error creating the authentication: %s`, err)
			}

			createdAuthentications = append(createdAuthentications, *authFixture)
		}
	}

	// Even though there are already other fixture authentications, we create another one to be perfectly sure that
	// there are more authentications on the database, other than the ones we are going to delete.
	authFixture := setUpValidAuthentication()
	authFixture.ResourceID = fixtures.TestSourceData[1].ID
	authFixture.ResourceType = "Source"

	err := authsDao.Create(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	// Call the function under test.
	deletedAuths, err := authsDao.BulkDelete(createdAuthentications)
	if err != nil {
		t.Errorf(`unexpected error when bulk deleting the authentications: %s`, err)
	}

	// Make sure that we haven't deleted more authentications than expected.
	{
		want := len(createdAuthentications)
		got := len(deletedAuths)
		if want != got {
			t.Errorf(`the deleted authentications don't match the created ones. Want "%d" auths deleted, got "%d"`, want, got)
		}
	}

	{
		for i := 0; i < len(createdAuthentications); i++ {
			{
				want := createdAuthentications[i].DbID
				got := deletedAuths[i].DbID
				if want != got {
					t.Errorf(`wrong authentication deleted. Want ID "%d", got ID "%d"`, want, got)
				}
			}
			{
				want := *createdAuthentications[i].Username
				got := *deletedAuths[i].Username
				if want != got {
					t.Errorf(`wrong authentication deleted. Want username "%s", got username "%s"`, want, got)
				}
			}
		}
	}

	// Check that the deleted resources come with the related tenant. This is necessary since otherwise the events will
	// not have the "tenant" key populated.
	for _, auth := range deletedAuths {
		want := fixtures.TestTenantData[0].ExternalTenant
		got := auth.Tenant.ExternalTenant

		if want != got {
			t.Errorf(`the authentication doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
		}
	}

	DropSchema("authentications_db")
}

// TestBulkDeleteRegression is a regression test for RHCLOUD-18907. Essentially, we were using Gorm's
// "Find(model, idSlice)" nicety, which automatically adds a "WHERE id IN ?" clause to the query. The gotcha was that
// if the slice is empty, Gorm just issues a regular "SELECT * FROM" without a "WHERE" clause, making it select the
// entire table. And since we were selecting the auths to be deleted, that behaviour is an issue.
//
// The test checks that in the case of receiving an empty slice, no authentications are deleted.
func TestBulkDeleteRegression(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)
	SwitchSchema("authentications_db")

	// How many authentications will we be creating per resource?
	maxAuthenticationsPerResource := 5

	authsDao := GetAuthenticationDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	for i := 0; i < maxAuthenticationsPerResource; i++ {
		authFixture := setUpValidAuthentication()
		authFixture.ResourceID = fixtures.TestSourceData[1].ID
		authFixture.ResourceType = "Source"
		// The fake username is used to make sure we are fetching the right authentication.
		fakeUsername := fmt.Sprintf("%s-%d", "Source", i)
		authFixture.Username = &fakeUsername

		err := authsDao.Create(authFixture)
		if err != nil {
			t.Errorf(`error creating the authentication: %s`, err)
		}
	}

	// Even though there are already other fixture authentications, we create another one to be perfectly sure that
	// there are more authentications on the database, other than the ones we are going to delete.
	authFixture := setUpValidAuthentication()
	authFixture.ResourceID = fixtures.TestSourceData[2].ID
	authFixture.ResourceType = "Source"

	err := authsDao.Create(authFixture)
	if err != nil {
		t.Errorf(`error creating the authentication: %s`, err)
	}

	var emptySlice = make([]model.Authentication, 0)

	// Call the function under test.
	deletedAuths, err := authsDao.BulkDelete(emptySlice)
	if err != nil {
		t.Errorf(`unexpected error when bulk deleting the authentications: %s`, err)
	}

	// Make sure that we haven't deleted more authentications than expected.
	{
		want := 0
		got := len(deletedAuths)
		if want != got {
			t.Errorf(`there shouldn't be any deleted authentications. Want "%d" auths deleted, got "%d"`, want, got)
		}
	}

	DropSchema("authentications_db")
}
