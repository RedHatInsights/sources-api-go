package statuslistener

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
)

var (
	// runningIntegration is used to skip integration tests if we're just running unit tests.
	runningIntegration = false

	endpointDao dao.EndpointDao
	sourceDao   dao.SourceDao
)

func TestMain(t *testing.M) {
	flags := testutils.ParseFlags()

	if flags.CreateDb {
		testutils.CreateTestDB()
	} else if flags.Integration {
		runningIntegration = true
		testutils.ConnectAndMigrateDB("status_listener")
		testutils.CreateFixtures()
	}

	code := t.Run()

	if flags.Integration {
		testutils.DropSchema("status_listener")
	}

	os.Exit(code)
}
