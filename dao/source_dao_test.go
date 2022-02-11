package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
)

var tenantId int64 = 1
var sourceDao = SourceDaoImpl{TenantID: &tenantId}

// TestSourcesGetRelatedRhcConnections tests whether the correct sources are fetched from the "related" function or not.
func TestSourcesGetRelatedRhcConnections(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures(RHC_CONNECTION_SCHEMA)

	sourceId := int64(1)

	rhcConnections, _, err := sourceDao.GetRelatedRhcConnectionsToId(&sourceId, 10, 0, nil)
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
