package dao

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
)

var sourceDao = sourceDaoImpl{
	TenantID: &fixtures.TestTenantData[0].Id,
}

// TestSourcesListForRhcConnections tests whether the correct sources are fetched from the related connection or not.
func TestSourcesListForRhcConnections(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema(RHC_CONNECTION_SCHEMA)

	rhcConnectionId := int64(1)

	sources, _, err := sourceDao.ListForRhcConnection(&rhcConnectionId, 10, 0, nil)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	// By taking a look at "fixtures/source_rhc_connection.go", we see that the "rchConnection" with ID 1 should have
	// two related sources connected. We use scoped variables so that  we can redeclare the "want" and "go" variables
	// with different types.
	{
		want := 2
		got := len(sources)
		if want != got {
			t.Errorf(`incorrect amount of related sources fetched. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := fixtures.TestSourceRhcConnectionData[0].SourceId
		got := sources[0].ID
		if want != got {
			t.Errorf(`incorrect related source fetched. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := fixtures.TestSourceRhcConnectionData[2].SourceId
		got := sources[1].ID
		if want != got {
			t.Errorf(`incorrect related source fetched. Want "%d", got "%d"`, want, got)
		}

	}

	DropSchema(RHC_CONNECTION_SCHEMA)
}

// testSource holds the test source that will be used through tests. It is saved in a variable to avoid having to write
// the full "fixtures..." thing every time.
var testSource = fixtures.TestSourceData[0]

// TestPausingSource checks whether the "paused_at" column gets successfully modified when pausing a source.
func TestPausingSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("pause_unpause")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &testSource.TenantID})
	err := sourceDao.Pause(testSource.ID)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	source, err := sourceDao.GetByIdWithPreload(&testSource.ID, "Applications")
	if err != nil {
		t.Errorf(`error fetching the source with its applications. Want nil error, got "%s"`, err)
	}

	want := time.Now()
	if !dateTimesAreSimilar(want, *source.PausedAt) {
		t.Errorf(`want "%s", got "%s"`, want, *source.PausedAt)
	}

	for _, app := range source.Applications {
		if !dateTimesAreSimilar(want, *app.PausedAt) {
			t.Errorf(`application not properly paused. Want "%s", got "%s"`, want, app.PausedAt)
		}
	}

	DropSchema("pause_unpause")
}

// TestResumingSource checks whether the "paused_at" column gets set as "NULL" when resuming a source.
func TestResumingSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("pause_unpause")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &testSource.TenantID})
	err := sourceDao.Unpause(fixtures.TestSourceData[0].ID)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	source, err := sourceDao.GetByIdWithPreload(&testSource.ID, "Applications")
	if err != nil {
		t.Errorf(`error fetching the source with its applications. Want nil error, got "%s"`, err)
	}

	var want *time.Time
	if want != source.PausedAt {
		t.Errorf(`want "%s", got "%s"`, want, source.PausedAt)
	}

	for _, app := range source.Applications {
		if app.PausedAt != nil {
			t.Errorf(`application not properly resumed. Want "%s", got "%s"`, want, app.PausedAt)
		}
	}

	DropSchema("pause_unpause")
}

// TestDeleteSource tests that a source gets correctly deleted, and its data returned.
func TestDeleteSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestSourceData[0].TenantID})

	source := fixtures.TestSourceData[0]
	// Set the ID to 0 to let GORM know it should insert a new source and not update an existing one.
	source.ID = 0
	// Set some data to compare the returned source.
	source.Name = "cool source"
	sourceUid := "abcde-fghij"
	source.Uid = &sourceUid

	// Create the test source.
	err := sourceDao.Create(&source)
	if err != nil {
		t.Errorf("error creating source: %s", err)
	}

	deletedSource, err := sourceDao.Delete(&source.ID)
	if err != nil {
		t.Errorf("error deleting an source: %s", err)
	}

	{
		want := source.ID
		got := deletedSource.ID

		if want != got {
			t.Errorf(`incorrect source deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	{
		want := source.Name
		got := deletedSource.Name

		if want != got {
			t.Errorf(`incorrect source deleted. Want "%s" in the name field, got "%s"`, want, got)
		}
	}

	DropSchema("delete")
}

// TestDeleteSourceNotExists tests that when a source that doesn't exist is tried to be deleted, an error is returned.
func TestDeleteSourceNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestSourceData[0].TenantID})

	nonExistentId := int64(12345)
	_, err := sourceDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DropSchema("delete")
}

// TestDeleteCascade is a long test function, but very simple in essence. Essentially what it does is:
//
//- It creates source with subresources (apps, endpoints, rhc-connections ...).
//- Cascade deletes the source with the function under test.
//- Checks that the deleted subresources and source are the ones that have been created in this very same test.
func TestDeleteCascade(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")
	tenantId := int64(1)

	// Create a new source fixture to avoid mixing the applications with the ones that already exist.
	fixtureSource := m.Source{
		Name:         "fixture-source",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     tenantId,
		Uid:          util.StringRef("new-shiny-source"),
	}

	// Try inserting the source in the database.
	sourceDaoParams := RequestParams{TenantID: &tenantId}
	sourceDao := GetSourceDao(&sourceDaoParams)
	err := sourceDao.Create(&fixtureSource)
	if err != nil {
		t.Errorf(`error creating a fixture source: %s`, err)
	}

	// Grab the DAOs which we will use to create the subresources.
	daoParams := RequestParams{TenantID: &tenantId}
	applicationAuthenticationDao := GetApplicationAuthenticationDao(&daoParams)
	applicationsDao := GetApplicationDao(&daoParams)
	authenticationDao := GetAuthenticationDao(&daoParams)
	endpointDao := GetEndpointDao(&fixtures.TestTenantData[0].Id)
	rhcConnectionsDao := GetRhcConnectionDao(&fixtures.TestTenantData[0].Id)

	// Create all the subresources.
	// Create the related application.
	app := m.Application{
		SourceID:          fixtureSource.ID,
		TenantID:          tenantId,
		ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
	}

	err = applicationsDao.Create(&app)
	if err != nil {
		t.Errorf(`error creating the application fixture: %s`, err)
	}

	// Create an authentication for application.
	authentication := setUpValidAuthentication()
	authentication.ResourceType = "Application"
	authentication.ResourceID = app.ID

	err = authenticationDao.Create(authentication)
	if err != nil {
		t.Errorf(`could not create the fixture authentication: %s`, err)
	}

	// Create the association between the application and its authentication.
	appAuth := m.ApplicationAuthentication{
		TenantID:          tenantId,
		ApplicationID:     app.ID,
		AuthenticationID:  authentication.DbID,
		AuthenticationUID: "authentication UID",
	}

	err = applicationAuthenticationDao.Create(&appAuth)
	if err != nil {
		t.Errorf(`could not create the fixture application authentication: %s`, err)
	}

	// Create the related endpoints.
	host := "test host"
	endpoint := m.Endpoint{
		Host:     &host,
		SourceID: fixtureSource.ID,
		TenantID: tenantId,
	}

	err = endpointDao.Create(&endpoint)
	if err != nil {
		t.Errorf(`error creating the endpoint fixture: %s`, err)
	}

	// Create the related rhcConnections.
	rhcId := "rhc connection id"
	rhcConnection := m.RhcConnection{
		RhcId:   rhcId,
		Sources: []m.Source{{ID: fixtureSource.ID}},
	}

	_, err = rhcConnectionsDao.Create(&rhcConnection)
	if err != nil {
		t.Errorf(`error creating the rhcConnection fixture: %s`, err)
	}

	// Call the function under test.
	deletedApplicationAuthentications, deletedApplications, deletedEndpoints, deletedRhcConnections, deletedSource, err := sourceDao.DeleteCascade(fixtureSource.ID)
	if err != nil {
		t.Errorf(`unexpected error when calling source delete cascade: %s`, err)
	}

	// Check that deleted app auth is not in the database
	{
		if len(deletedApplicationAuthentications) != 1 {
			t.Errorf("different count of app auths deleted, want %d, deleted %d", 1, len(deletedApplicationAuthentications))
		}
		id := deletedApplicationAuthentications[0].ID
		_, err = applicationAuthenticationDao.GetById(&id)
		if !errors.Is(err, util.ErrNotFoundEmpty) {
			t.Errorf("Expected not found error, got %s", err)
		}
	}

	// Check that deleted app is not in the database
	{
		if len(deletedApplications) != 1 {
			t.Errorf("different count of apps deleted, want %d, deleted %d", 1, len(deletedApplications))
		}
		id := deletedApplications[0].ID
		_, err = applicationsDao.GetById(&id)
		if !errors.Is(err, util.ErrNotFoundEmpty) {
			t.Errorf("Expected not found error, got %s", err)
		}
	}

	// Check that deleted endpoint is not in the database
	{
		if len(deletedEndpoints) != 1 {
			t.Errorf("different count of apps deleted, want %d, deleted %d", 1, len(deletedEndpoints))
		}
		id := deletedEndpoints[0].ID
		_, err = endpointDao.GetById(&id)
		if !errors.Is(err, util.ErrNotFoundEmpty) {
			t.Errorf("Expected not found error, got %s", err)
		}
	}

	// Check that deleted rhc connection is not in the database
	{
		if len(deletedRhcConnections) != 1 {
			t.Errorf("different count of apps deleted, want %d, deleted %d", 1, len(deletedRhcConnections))
		}
		id := deletedRhcConnections[0].ID
		_, err = rhcConnectionsDao.GetById(&id)
		if !errors.Is(err, util.ErrNotFoundEmpty) {
			t.Errorf("Expected not found error, got %s", err)
		}
	}

	// Check that deleted source is not in the database
	{
		id := deletedSource.ID
		_, err = sourceDao.GetById(&id)
		if !errors.Is(err, util.ErrNotFoundEmpty) {
			t.Errorf("Expected not found error, got %s", err)
		}
	}

	// Check that created authentication was not deleted
	id := fmt.Sprintf("%d", authentication.DbID)
	authOut, err := authenticationDao.GetById(id)
	if err != nil {
		t.Error(err)
	}
	if authOut.DbID != authentication.DbID {
		t.Errorf("ghost infected the return")
	}
	if authOut.ResourceType != authentication.ResourceType {
		t.Errorf("ghost infected the return")
	}
	if authOut.ResourceID != authentication.ResourceID {
		t.Errorf("ghost infected the return")
	}
	if authOut.SourceID != authentication.SourceID {
		t.Errorf("ghost infected the return")
	}

	// Delete the authentication
	_, err = authenticationDao.Delete(id)
	if err != nil {
		t.Error(err)
	}

	// Check that the authentication is deleted
	_, err = authenticationDao.GetById(id)
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf("Expected not found error, got %s", err)
	}

	DropSchema("delete")
}

// TestDeleteCascadeSourceWithoutSubresources tests the deletion of source that doesn't have subresources
func TestDeleteCascadeSourceWithoutSubresources(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")
	tenantId := int64(1)

	// Create a new source fixture to avoid mixing the applications with the ones that already exist.
	fixtureSource := m.Source{
		Name:         "fixture-source",
		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
		TenantID:     tenantId,
		Uid:          util.StringRef("new-shiny-source"),
	}

	// Try inserting the source in the database.
	sourceDaoParams := RequestParams{TenantID: &tenantId}
	sourceDao := GetSourceDao(&sourceDaoParams)
	err := sourceDao.Create(&fixtureSource)
	if err != nil {
		t.Errorf(`error creating a fixture source: %s`, err)
	}

	// Check that the only deleted resource should be the source itself, since it doesn't have any subresources.
	applicationAuthentications, applications, endpoints, rhcConnections, deletedSource, err := sourceDao.DeleteCascade(fixtureSource.ID)
	if err != nil {
		t.Errorf(`unexpected error received when cascade deleting the source: %s`, err)
	}

	// Double-check the "deleted" resources and the source itself.
	{
		want := 0
		got := len(applicationAuthentications)
		if want != got {
			t.Errorf(`unexpected application authentications deleted. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := 0
		got := len(applications)
		if want != got {
			t.Errorf(`unexpected applications deleted. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := 0
		got := len(endpoints)
		if len(endpoints) != 0 {
			t.Errorf(`unexpected endpoints deleted. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := 0
		got := len(rhcConnections)
		if len(rhcConnections) != 0 {
			t.Errorf(`unexpected rhcConnections deleted. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := fixtureSource.ID
		got := deletedSource.ID
		if want != got {
			t.Errorf(`wrong source deleted. Want source with ID "%d", got ID "%d"`, want, got)
		}
	}

	DropSchema("delete")
}

// TestSourceExists tests whether the function exists returns true when the given source exists.
func TestSourceExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	got, err := sourceDao.Exists(fixtures.TestSourceData[0].ID)
	if err != nil {
		t.Errorf(`unexpected error when checking that the source exists: %s`, err)
	}

	if !got {
		t.Errorf(`the source does exist but the "Exist" function returns otherwise. Want "true", got "%t"`, got)
	}

	DropSchema("exists")
}

// TestSourceNotExists tests whether the function exists returns false when the given source does not exist.
func TestSourceNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	sourceDao := GetSourceDao(&RequestParams{TenantID: &fixtures.TestTenantData[0].Id})

	got, err := sourceDao.Exists(12345)
	if err != nil {
		t.Errorf(`unexpected error when checking that the source exists: %s`, err)
	}

	if got {
		t.Errorf(`the source doesn't exist but the "Exist" function returns otherwise. Want "false", got "%t"`, got)
	}

	DropSchema("exists")
}

// TestSourceSubcollectionListWithOffsetAndLimit tests that SubCollectionList() in source dao returns
//  correct count value and correct count of returned objects
func TestSourceSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	sourceTypeId := fixtures.TestSourceTypeData[0].Id

	var wantCount int64
	for _, i := range fixtures.TestSourceData {
		if i.SourceTypeID == sourceTypeId {
			wantCount++
		}
	}

	for _, d := range fixtures.TestDataOffsetLimit {
		sources, gotCount, err := sourceDao.SubCollectionList(m.SourceType{Id: sourceTypeId}, d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the sources: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of sources, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(sources)
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

// TestSourceListOffsetAndLimit tests that List() in source dao returns correct count value
// and correct count of returned objects
func TestSourceListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	wantCount := int64(len(fixtures.TestSourceData))

	for _, d := range fixtures.TestDataOffsetLimit {
		sources, gotCount, err := sourceDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the sources: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of sources, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(sources)
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

// TestSourceListInternalOffsetAndLimit tests that ListInternal() in source dao returns correct count value
// and correct count of returned objects
func TestSourceListInternalOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	wantCount := int64(len(fixtures.TestSourceData))

	for _, d := range fixtures.TestDataOffsetLimit {
		sources, gotCount, err := sourceDao.ListInternal(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the sources: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of sources, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(sources)
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

// TestSourceListForRhcConnectionWithOffsetAndLimit tests that ListForRhcConnection() in source dao returns
//  correct count value and correct count of returned objects
func TestSourceListForRhcConnectionWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	rhcConnectionId := fixtures.TestRhcConnectionData[0].ID

	var wantCount int64
	for _, i := range fixtures.TestSourceRhcConnectionData {
		if i.RhcConnectionId == rhcConnectionId {
			wantCount++
		}
	}

	for _, d := range fixtures.TestDataOffsetLimit {
		sources, gotCount, err := sourceDao.ListForRhcConnection(&rhcConnectionId, d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the sources: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of sources, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(sources)
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

func TestSourceListUserOwnership(t *testing.T) {
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
	sourceDaoWithUser := GetSourceDao(requestParams)

	sources, _, err := sourceDaoWithUser.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	var sourcesIDs []int64
	for _, source := range sources {
		sourcesIDs = append(sourcesIDs, source.ID)
	}

	var expectedSourcesIDs []int64
	for _, appAuth := range recordsWithUserID.Sources {
		expectedSourcesIDs = append(expectedSourcesIDs, appAuth.ID)
	}

	for _, appAuth := range recordsWithoutUserID.Sources {
		expectedSourcesIDs = append(expectedSourcesIDs, appAuth.ID)
	}

	if !cmp.Equal(sourcesIDs, expectedSourcesIDs) {
		t.Errorf("Expected application authentication IDS %v are not same with obtained ids: %v", expectedSourcesIDs, sourcesIDs)
	}

	userWithoutOwnRecords, err := CreateUserForUserID(userIDWithoutOwnRecords, user.TenantID)
	if err != nil {
		t.Errorf(`unable to create user: %v`, err)
	}

	requestParams = &RequestParams{TenantID: &user.TenantID, UserID: &userWithoutOwnRecords.Id}
	sourceDaoWithUser = GetSourceDao(requestParams)

	sources, _, err = sourceDaoWithUser.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application authentications: %s`, err)
	}

	sourcesIDs = []int64{}
	for _, source := range sources {
		sourcesIDs = append(sourcesIDs, source.ID)
	}

	expectedSourcesIDs = []int64{}
	for _, source := range recordsWithoutUserID.Sources {
		expectedSourcesIDs = append(expectedSourcesIDs, source.ID)
	}

	if !cmp.Equal(sourcesIDs, expectedSourcesIDs) {
		t.Errorf("Expected application authentication IDS %v are not same with obtained ids: %v", expectedSourcesIDs, sourcesIDs)
	}

	DropSchema(schema)
}
