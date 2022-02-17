package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
)

var sourceDao = SourceDaoImpl{
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
