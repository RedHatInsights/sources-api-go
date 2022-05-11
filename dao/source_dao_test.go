package dao

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
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

	sourceDao := GetSourceDao(&testSource.TenantID)
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

	sourceDao := GetSourceDao(&testSource.TenantID)
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

	sourceDao := GetSourceDao(&fixtures.TestSourceData[0].TenantID)

	source := fixtures.TestSourceData[0]
	// Set the ID to 0 to let GORM know it should insert a new source and not update an existing one.
	source.ID = 0
	// Set some data to compare the returned source.
	source.Name = "cool source"

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

	sourceDao := GetSourceDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := sourceDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DropSchema("delete")
}

// TestDeleteCascade is a long test function, but very simple in essence. Essentially what it does is:
//
// - It checks that when a source with no subresources is deleted, only that particular source is deleted.
// - It creates N applications, endpoints and rhcConnections for a fixture source.
// - Cascade deletes the source with the function under test.
// - Checks that the deleted subresources and source are the ones that have been created in this very same test.
// func TestDeleteCascade(t *testing.T) {
// 	testutils.SkipIfNotRunningIntegrationTests(t)
// 	SwitchSchema("delete")

// 	// Create a new source fixture to avoid mixing the applications with the ones that already exist.
// 	fixtureSource := m.Source{
// 		Name:         "fixture-source",
// 		SourceTypeID: fixtures.TestSourceTypeData[0].Id,
// 		TenantID:     fixtures.TestTenantData[0].Id,
// 		Uid:          util.StringRef("new-shiny-source"),
// 	}

// 	// Try inserting the source in the database.
// 	sourceDao := GetSourceDao(&fixtures.TestTenantData[0].Id)
// 	err := sourceDao.Create(&fixtureSource)
// 	if err != nil {
// 		t.Errorf(`error creating a fixture source: %s`, err)
// 	}

// 	// Check that the only deleted resource should be the source itself, since it doesn't have any subresources.
// 	applicationAuthentications, applications, endpoints, rhcConnections, deletedSource, err := sourceDao.DeleteCascade(fixtureSource.ID)
// 	if err != nil {
// 		t.Errorf(`unexpected error received when cascade deleting the source: %s`, err)
// 	}

// 	// Double-check the "deleted" resources and the source itself.
// 	{
// 		want := 0
// 		got := len(applicationAuthentications)
// 		if want != got {
// 			t.Errorf(`unexpected application authentications deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}

// 	{
// 		want := 0
// 		got := len(applications)
// 		if want != got {
// 			t.Errorf(`unexpected applications deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}
// 	{
// 		want := 0
// 		got := len(endpoints)
// 		if len(endpoints) != 0 {
// 			t.Errorf(`unexpected endpoints deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}

// 	{
// 		want := 0
// 		got := len(rhcConnections)
// 		if len(rhcConnections) != 0 {
// 			t.Errorf(`unexpected rhcConnections deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}

// 	{
// 		want := fixtureSource.ID
// 		got := deletedSource.ID
// 		if want != got {
// 			t.Errorf(`wrong source deleted. Want source with ID "%d", got ID "%d"`, want, got)
// 		}
// 	}

// 	// Reinsert the source.
// 	fixtureSource.ID = 0
// 	err = sourceDao.Create(&fixtureSource)
// 	if err != nil {
// 		t.Errorf(`error creating a fixture source: %s`, err)
// 	}

// 	// Grab the DAOs which we will use to create the subresources.
// 	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestTenantData[0].Id)
// 	applicationsDao := GetApplicationDao(&fixtures.TestTenantData[0].Id)
// 	authenticationDao := GetAuthenticationDao(&fixtures.TestTenantData[0].Id)
// 	endpointDao := GetEndpointDao(&fixtures.TestTenantData[0].Id)
// 	rhcConnectionsDao := GetRhcConnectionDao(&fixtures.TestTenantData[0].Id)

// 	// Establish a maximum number of subresources we want to insert.
// 	maximumSubresourcesToCreate := 5

// 	// We want the created subresources so that we can compare them later to the deleted ones.
// 	var createdApplications []m.Application
// 	var createdApplicationAuthentications []m.ApplicationAuthentication
// 	var createdEndpoints []m.Endpoint
// 	var createdRhcConnections []m.RhcConnection

// 	// Create all the subresources.
// 	for i := 0; i < maximumSubresourcesToCreate; i++ {
// 		// Create the related applications.
// 		extra := fmt.Sprintf(`{"idx": "%d"}`, i)
// 		app := m.Application{
// 			Extra:             []byte(extra),
// 			SourceID:          fixtureSource.ID,
// 			TenantID:          fixtureSource.TenantID,
// 			ApplicationTypeID: fixtures.TestApplicationTypeData[0].Id,
// 		}

// 		err := applicationsDao.Create(&app)
// 		if err != nil {
// 			t.Errorf(`error creating the application fixture: %s`, err)
// 		}

// 		createdApplications = append(createdApplications, app)

// 		// Create one authentication per application.
// 		authentication := setUpValidAuthentication()
// 		authentication.ResourceType = "Application"
// 		authentication.ResourceID = app.ID

// 		err = authenticationDao.Create(authentication)
// 		if err != nil {
// 			t.Errorf(`could not create the fixture authentication: %s`, err)
// 		}

// 		// Create the association between the application and its authentication.
// 		appAuth := m.ApplicationAuthentication{
// 			TenantID:          fixtures.TestTenantData[0].Id,
// 			ApplicationID:     app.ID,
// 			AuthenticationID:  authentication.DbID,
// 			AuthenticationUID: fmt.Sprintf("%d", i),
// 		}

// 		err = applicationAuthenticationDao.Create(&appAuth)
// 		if err != nil {
// 			t.Errorf(`could not create the fixture application authentication: %s`, err)
// 		}
// 		createdApplicationAuthentications = append(createdApplicationAuthentications, appAuth)

// 		// Create the related endpoints.
// 		host := fmt.Sprintf(`domain%d.com`, i)
// 		endpoint := m.Endpoint{
// 			Host:     &host,
// 			SourceID: fixtureSource.ID,
// 			TenantID: fixtureSource.TenantID,
// 		}

// 		err = endpointDao.Create(&endpoint)
// 		if err != nil {
// 			t.Errorf(`error creating the endpoint fixture: %s`, err)
// 		}

// 		createdEndpoints = append(createdEndpoints, endpoint)

// 		// Create the related rhcConnections.
// 		rhcId := fmt.Sprintf(`rhc-id-%d`, i)
// 		rhcConnection := m.RhcConnection{
// 			RhcId:   rhcId,
// 			Sources: []m.Source{{ID: fixtureSource.ID}},
// 		}

// 		_, err = rhcConnectionsDao.Create(&rhcConnection)
// 		if err != nil {
// 			t.Errorf(`error creating the rhcConnection fixture: %s`, err)
// 		}

// 		createdRhcConnections = append(createdRhcConnections, rhcConnection)
// 	}

// 	// Call the function under test.
// 	deletedApplicationAuthentications, deletedApplications, deletedEndpoints, deletedRhcConnections, deletedSource, err := sourceDao.DeleteCascade(fixtureSource.ID)
// 	if err != nil {
// 		t.Errorf(`unexpected error when calling source delete cascade: %s`, err)
// 	}

// 	// Count the application authentications from the given source, to check that they were deleted.
// 	var appAuthsCount int64
// 	err = DB.
// 		Debug().
// 		Model(m.ApplicationAuthentication{}).
// 		Joins(`INNER JOIN "applications" ON "application_authentications"."application_id" = "applications"."id"`).
// 		Where(`applications.source_id = ?`, fixtureSource.ID).
// 		Where(`applications.tenant_id = ?`, fixtures.TestTenantData[0].Id).
// 		Count(&appAuthsCount).
// 		Error

// 	if err != nil {
// 		t.Errorf(`error counting the application authentications related to the source: %s`, err)
// 	}

// 	// Check if the application authentications were deleted or not.
// 	{
// 		want := int64(0)
// 		got := appAuthsCount
// 		if want != got {
// 			t.Errorf(`the application authentications were not deleted. Want a count of "%d", got "%d"`, want, got)
// 		}
// 	}

// 	// Check that the expected applicationAuthentications were deleted.
// 	for i := 0; i < maximumSubresourcesToCreate; i++ {
// 		{
// 			want := createdApplicationAuthentications[i].ID
// 			got := deletedApplicationAuthentications[i].ID

// 			if want != got {
// 				t.Errorf(`unexpected application authentication deleted. Want application with ID "%d", got ID "%d"`, want, got)
// 			}
// 		}

// 		{
// 			want := createdApplicationAuthentications[i].ApplicationID
// 			got := deletedApplicationAuthentications[i].ApplicationID

// 			if want != got {
// 				t.Errorf(`unexpected application authentication deleted. Want application authentication with application ID "%d", got "%d"`, want, got)
// 			}
// 		}
// 	}

// 	// Count the applications from the given source, to check that they were deleted.
// 	var appCount int64
// 	err = DB.
// 		Debug().
// 		Model(m.Application{}).
// 		Where("source_id = ?", fixtureSource.ID).
// 		Where("tenant_id = ?", fixtures.TestTenantData[0].Id).
// 		Count(&appCount).
// 		Error

// 	if err != nil {
// 		t.Errorf(`error counting the applications related to the source: %s`, err)
// 	}

// 	// Check if the applications were deleted or not.
// 	{
// 		want := int64(0)
// 		got := appCount
// 		if want != got {
// 			t.Errorf(`the applications were not deleted. Want a count of "%d", got "%d"`, want, got)
// 		}
// 	}

// 	// Check that the expected applications were deleted.
// 	for i := 0; i < maximumSubresourcesToCreate; i++ {
// 		{
// 			want := createdApplications[i].ID
// 			got := deletedApplications[i].ID

// 			if want != got {
// 				t.Errorf(`unexpected application deleted. Want application with ID "%d", got ID "%d"`, want, got)
// 			}
// 		}

// 		{
// 			want := createdApplications[i].Extra
// 			got := deletedApplications[i].Extra

// 			if !bytes.Equal(want, got) {
// 				t.Errorf(`unexpected application deleted. Want application with extra "%s", got extra "%s"`, want, got)
// 			}
// 		}
// 	}

// 	// Count the endpoints from the given source, to check that they were deleted.
// 	var endpointCount int64
// 	err = DB.
// 		Debug().
// 		Model(m.Endpoint{}).
// 		Where("source_id = ?", fixtureSource.ID).
// 		Where("tenant_id = ?", fixtures.TestTenantData[0].Id).
// 		Count(&endpointCount).
// 		Error

// 	if err != nil {
// 		t.Errorf(`error counting the endpoints related to the source: %s`, err)
// 	}

// 	// Check if the endpoints were deleted or not.
// 	{
// 		want := int64(0)
// 		got := endpointCount
// 		if want != got {
// 			t.Errorf(`the endpoints were not deleted. Want a count of "%d", got "%d"`, want, got)
// 		}
// 	}

// 	{
// 		want := len(createdEndpoints)
// 		got := len(deletedEndpoints)

// 		if want != got {
// 			t.Errorf(`unexpected amount of endpoints deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}

// 	// Check that the expected endpoints were deleted.
// 	for i := 0; i < maximumSubresourcesToCreate; i++ {
// 		{
// 			want := createdEndpoints[i].ID
// 			got := deletedEndpoints[i].ID

// 			if want != got {
// 				t.Errorf(`unexpected endpoint deleted. Want endpoint with ID "%d", got ID "%d"`, want, got)
// 			}
// 		}

// 		{
// 			want := *createdEndpoints[i].Host
// 			got := *deletedEndpoints[i].Host

// 			if want != got {
// 				t.Errorf(`unexpected endpoint deleted. Want endpoint with host "%s", got host "%s"`, want, got)
// 			}
// 		}
// 	}

// 	// Count the rhcConnections from the given source, to check that they were deleted.
// 	var rhcConnectionsCount int64
// 	err = DB.
// 		Debug().
// 		Model(m.RhcConnection{}).
// 		Joins(`INNER JOIN "source_rhc_connections" "sr" ON "rhc_connections"."id" = "sr"."rhc_connection_id"`).
// 		Where(`"sr"."source_id" = ?`, fixtureSource.ID).
// 		Where(`"sr"."tenant_id" = ?`, fixtures.TestTenantData[0].Id).
// 		Count(&rhcConnectionsCount).
// 		Error

// 	if err != nil {
// 		t.Errorf(`error counting the rhcConnections related to the source: %s`, err)
// 	}

// 	// Check if the rhcConnections were deleted or not.
// 	{
// 		want := int64(0)
// 		got := rhcConnectionsCount
// 		if want != got {
// 			t.Errorf(`the rhcConnections were not deleted. Want a count of "%d", got "%d"`, want, got)
// 		}
// 	}

// 	{
// 		want := len(createdRhcConnections)
// 		got := len(deletedRhcConnections)

// 		if want != got {
// 			t.Errorf(`unexpected amount of rhcConnections deleted. Want "%d", got "%d"`, want, got)
// 		}
// 	}

// 	// Check that the expected rhcConnections were deleted.
// 	for i := 0; i < maximumSubresourcesToCreate; i++ {
// 		{
// 			want := createdRhcConnections[i].ID
// 			got := deletedRhcConnections[i].ID

// 			if want != got {
// 				t.Errorf(`unexpected rhcConnection deleted. Want rhcConnection with ID "%d", got ID "%d"`, want, got)
// 			}
// 		}

// 		{
// 			want := createdRhcConnections[i].RhcId
// 			got := deletedRhcConnections[i].RhcId

// 			if want != got {
// 				t.Errorf(`unexpected rhcConnection deleted. Want rhcConnection with RhcId "%s", got RhcId "%s"`, want, got)
// 			}
// 		}
// 	}

// 	// Try to fetch the deleted source.
// 	var deletedSourceCheck *m.Source
// 	err = DB.
// 		Debug().
// 		Model(m.Source{}).
// 		Where(`id = ?`, fixtureSource.ID).
// 		Where(`tenant_id = ?`, fixtures.TestTenantData[0].Id).
// 		Find(&deletedSourceCheck).
// 		Error

// 	if err != nil {
// 		t.Errorf(`unexpected error: %s`, err)
// 	}

// 	// Check that the expected source was deleted.
// 	if deletedSourceCheck.ID != 0 {
// 		t.Errorf(`unexpected source fetched. It should be deleted, but this source was fetched: %v`, deletedSourceCheck)
// 	}

// 	{
// 		want := fixtureSource.Name
// 		got := deletedSource.Name

// 		if want != got {
// 			t.Errorf(`wrong source deleted. Want source with name "%s", got "%s"`, want, got)
// 		}
// 	}

// 	{
// 		want := fixtureSource.ID
// 		got := deletedSource.ID

// 		if want != got {
// 			t.Errorf(`wrong source deleted. Want source with id "%d", got "%d"`, want, got)
// 		}
// 	}

// 	// Check that the deleted resources come with the related tenant. This is necessary since otherwise the events will
// 	// not have the "tenant" key populated.
// 	for _, applicationAuthentication := range deletedApplicationAuthentications {
// 		want := fixtures.TestTenantData[0].ExternalTenant
// 		got := applicationAuthentication.Tenant.ExternalTenant

// 		if want != got {
// 			t.Errorf(`the application authentication doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
// 		}
// 	}

// 	for _, application := range deletedApplications {
// 		want := fixtures.TestTenantData[0].ExternalTenant
// 		got := application.Tenant.ExternalTenant

// 		if want != got {
// 			t.Errorf(`the application doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
// 		}
// 	}

// 	for _, endpoint := range deletedEndpoints {
// 		want := fixtures.TestTenantData[0].ExternalTenant
// 		got := endpoint.Tenant.ExternalTenant

// 		if want != got {
// 			t.Errorf(`the endpoint doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
// 		}
// 	}

// 	want := fixtures.TestTenantData[0].ExternalTenant
// 	got := deletedSource.Tenant.ExternalTenant

// 	if want != got {
// 		t.Errorf(`the source doesn't come with the related tenant. Want external tenant "%s", got "%s"`, want, got)
// 	}

// 	DropSchema("delete")
// }

// TestSourceExists tests whether the function exists returns true when the given source exists.
func TestSourceExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	sourceDao := GetSourceDao(&fixtures.TestTenantData[0].Id)

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

	sourceDao := GetSourceDao(&fixtures.TestTenantData[0].Id)

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
