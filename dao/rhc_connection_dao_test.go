package dao

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const RHC_CONNECTION_SCHEMA = "rhc_connection"

var tenantId = fixtures.TestTenantData[0].Id
var rhcConnectionDao = rhcConnectionDaoImpl{
	TenantID: &tenantId,
}

// setUpValidRhcConnection returns a valid RhcConnection object.
func setUpValidRhcConnection() *model.RhcConnection {
	return &model.RhcConnection{
		RhcId: "rhcIdUuid",
		Extra: []byte(`{"hello": "world"}`),
		AvailabilityStatus: model.AvailabilityStatus{
			AvailabilityStatus: "available",
		},
		Sources: []model.Source{
			{
				ID: fixtures.TestSourceData[0].ID,
			},
		},
	}
}

// TestRhcConnectionCreate tests that when proper input is provided, the "Create" function creates the proper row and
// associated row in the "rhc_connections" and "source_rhc_connections" tables.
func TestRhcConnectionCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	want := setUpValidRhcConnection()
	got, err := rhcConnectionDao.Create(want)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if 0 == got.ID {
		t.Errorf(`want non zero ID, got "%d"`, got.ID)
	}

	if want.RhcId != got.RhcId {
		t.Errorf(`want "%s", got "%s"`, want.RhcId, got.RhcId)
	}

	if !bytes.Equal(want.Extra, got.Extra) {
		t.Errorf(`ẁant "%s", got "%s"`, want.Extra, got.Extra)
	}

	if want.AvailabilityStatus.AvailabilityStatus != got.AvailabilityStatus.AvailabilityStatus {
		t.Errorf(
			`want "%s", got "%s"`,
			want.AvailabilityStatus.AvailabilityStatus,
			got.AvailabilityStatus.AvailabilityStatus,
		)
	}

	var gotJoinTable model.SourceRhcConnection
	err = DB.Debug().
		Model(&model.SourceRhcConnection{}).
		Where(`rhc_connection_id = ?`, got.ID).
		Find(&gotJoinTable).
		Error

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if gotJoinTable.RhcConnectionId != got.ID {
		t.Errorf(`want "%d", got "%d"`, got.ID, gotJoinTable.RhcConnectionId)
	}

	if want.Sources[0].ID != gotJoinTable.SourceId {
		t.Errorf(`want "%d", got "%d"`, got.ID, gotJoinTable.SourceId)
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionCreateExistingSourceDifferentTenant tests that when querying for a source from a tenant that is not
// related to that source, the DAO throws a "not found" error. This is because people from other tenants shouldn't be
// able to link connections to sources from other tenants.
func TestRhcConnectionCreateExistingSourceDifferentTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	rhcConnection := setUpValidRhcConnection()

	// Set up a different tenant, which should make the "find source by source ID and Tenant ID" return a not found
	// error
	invalidTenantId := int64(12345)
	rhcConnectionDao.TenantID = &invalidTenantId
	_, got := rhcConnectionDao.Create(rhcConnection)

	want := "source not found"
	if want != got.Error() {
		t.Errorf(`want "%s", got "%s"`, want, got)
	}

	// Set the tenant back to its original value
	rhcConnectionDao.TenantID = &tenantId
	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionCreateExisting tests that when an already existing "RhcConnection" is given to the "Create"
// function, along with a valid and unique source ID on the "source_rhc_connections" table, the function doesn't return
// an error.
func TestRhcConnectionCreateExisting(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	want := setUpValidRhcConnection()
	want.RhcId = fixtures.TestRhcConnectionData[2].RhcId
	want.Sources[0].ID = fixtures.TestSourceData[0].ID

	got, err := rhcConnectionDao.Create(want)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if 0 == got.ID {
		t.Errorf(`want non zero ID, got "%d"`, got.ID)
	}

	if want.RhcId != got.RhcId {
		t.Errorf(`want "%s", got "%s"`, want.RhcId, got.RhcId)
	}

	if !bytes.Equal(want.Extra, got.Extra) {
		t.Errorf(`ẁant "%s", got "%s"`, want.Extra, got.Extra)
	}

	if want.AvailabilityStatus.AvailabilityStatus != got.AvailabilityStatus.AvailabilityStatus {
		t.Errorf(
			`want "%s", got "%s"`,
			want.AvailabilityStatus.AvailabilityStatus,
			got.AvailabilityStatus.AvailabilityStatus,
		)
	}

	var gotJoinTable = make([]model.SourceRhcConnection, 0, 2)
	err = DB.Debug().
		Model(&model.SourceRhcConnection{}).
		Where(`rhc_connection_id = ?`, got.ID).
		Find(&gotJoinTable).
		Error

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	// In this check we can simply grab the first ID of the RhcConnection, since all the results belong to the same
	// RhConnection.
	if gotJoinTable[0].RhcConnectionId != got.ID {
		t.Errorf(`want "%d", got "%d"`, got.ID, gotJoinTable[0].RhcConnectionId)
	}

	// We have to loop through the fetched sources' ids, since it's a many-to-many relationship.
	var found = false
	for _, joinTableRow := range gotJoinTable {
		if want.Sources[0].ID == joinTableRow.SourceId {
			found = true
		}
	}

	if !found {
		t.Errorf(`want to find "%d" in "%v", but it was not found`, want.Sources[0].ID, gotJoinTable)
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionCreateSourceNotExists tests that a proper error is returned when a non-existing related source is
// given.
func TestRhcConnectionCreateSourceNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	// Modify the valid object to make it point to a non-existing source.
	rhcConnection := setUpValidRhcConnection()
	rhcConnection.Sources[0].ID = 12345

	_, err := rhcConnectionDao.Create(rhcConnection)

	if err == nil {
		t.Errorf("want non nil error, got nil error")
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`want "%s" type, got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionCreateAlreadyExistingAssociation tests that when an error is returned when an already existing
// association between the source and the rhcConnection exists in the join table.
func TestRhcConnectionCreateAlreadyExistingAssociation(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	rhcConnection := setUpValidRhcConnection()

	_, err := rhcConnectionDao.Create(rhcConnection)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	want := "connection already exists"

	_, err = rhcConnectionDao.Create(rhcConnection)
	if !strings.Contains(err.Error(), want) {
		t.Errorf(`want "%s", got "%s"`, want, err)
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionDelete tests that when an rhcConnection is deleted, its associations in the join table are also
// deleted.
func TestRhcConnectionDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	_, err := rhcConnectionDao.Delete(&fixtures.TestRhcConnectionData[0].ID)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	var rhcConnectionExists bool
	err = DB.Debug().
		Model(&model.RhcConnection{}).
		Select(`1`).
		Where(`id = ?`, fixtures.TestRhcConnectionData[0].ID).
		Find(&rhcConnectionExists).
		Error

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	if rhcConnectionExists {
		t.Errorf(`want "rhcConnection" deleted, data found`)
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionDeleteNotFound tests that when a non-existent ID is given to the delete function, a "not found"
// error is returned.
func TestRhcConnectionDeleteNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	nonExistentId := int64(12345)

	_, err := rhcConnectionDao.Delete(&nonExistentId)

	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`want "%s" type, got "%s"`, reflect.TypeOf(util.ErrNotFoundEmpty), reflect.TypeOf(err))
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionListForSources tests whether the correct connections are fetched from the related source or not.
func TestRhcConnectionListForSources(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	sourceId := int64(1)

	rhcConnections, _, err := rhcConnectionDao.ListForSource(&sourceId, 10, 0, nil)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	// By taking a look at "fixtures/source_rhc_connection.go", we see that the "source" with ID 1 should have
	// two related rhc connections. We use scoped variables so that  we can redeclare the "want" and "go" variables
	// with different types.
	{
		want := 2
		got := len(rhcConnections)
		if want != got {
			t.Errorf(`incorrect amount of related rhc connections fetched. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := fixtures.TestSourceRhcConnectionData[0].RhcConnectionId
		got := rhcConnections[0].ID
		if want != got {
			t.Errorf(`incorrect related rhc connection fetched. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := fixtures.TestSourceRhcConnectionData[1].RhcConnectionId
		got := rhcConnections[1].ID
		if want != got {
			t.Errorf(`incorrect related rhc connection fetched. Want "%d", got "%d"`, want, got)
		}

	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// TestRhcConnectionRowsClosed is a regression test for https://issues.redhat.com/browse/RHCLOUD-18192. It tests that
// when there are no rows to process from a result set, a proper response is returned instead of a "rows are closed"
// error.
func TestRhcConnectionRowsClosed(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	// Find all the connections that we will remove from the DB.
	dbRhcConnections := make([]model.RhcConnection, 0)
	err := DB.Debug().
		Model(&model.RhcConnection{}).
		Find(&dbRhcConnections).
		Error

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	// Remove each connection so we can simulate a "rows are closed" situation, where the ".Next" function returns a
	// "false" value.
	for _, conn := range dbRhcConnections {
		err = DB.Debug().
			Delete(conn).
			Error

		if err != nil {
			t.Errorf(`want nil error, got "%s"`, err)
		}
	}

	// The result should be an empty slice of connections, with no error thrown.
	rhcConnections, _, err := rhcConnectionDao.List(10, 0, nil)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	want := 0
	got := len(rhcConnections)
	if want != got {
		t.Errorf(`want "%d" connections from the database, got "%d"`, want, got)
	}

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}
