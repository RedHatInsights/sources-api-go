package dao

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
)

var sourceDao = sourceDaoImpl{
	TenantID: &fixtures.TestTenantData[0].Id,
}

// TestSourcesListForRhcConnections tests whether the correct sources are fetched from the related connection or not.
func TestSourcesListForRhcConnections(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

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

	DoneWithFixtures(RHC_CONNECTION_SCHEMA)
}

// testSource holds the test source that will be used through tests. It is saved in a variable to avoid having to write
// the full "fixtures..." thing every time.
var testSource = fixtures.TestSourceData[0]

// TestPausingSource checks whether the "paused_at" column gets successfully modified when pausing a source.
func TestPausingSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("pause_unpause")

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

	DoneWithFixtures("pause_unpause")
}

// TestResumingSource checks whether the "paused_at" column gets set as "NULL" when resuming a source.
func TestResumingSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	CreateFixtures("pause_unpause")

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

	DoneWithFixtures("pause_unpause")
}

// TestDeleteSource tests that a source gets correctly deleted, and its data returned.
func TestDeleteSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

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

	DoneWithFixtures("delete")
}

// TestDeleteSourceNotExists tests that when a source that doesn't exist is tried to be deleted, an error is returned.
func TestDeleteSourceNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	sourceDao := GetSourceDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := sourceDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DoneWithFixtures("delete")
}
