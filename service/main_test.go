package service

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
)

var (
	// runningIntegration is used to skip integration tests if we're just running unit tests.
	runningIntegration = false

	sourceDao dao.SourceDao
)

func TestMain(t *testing.M) {
	createDb, integration := testutils.ParseFlags()

	if createDb {
		testutils.CreateTestDB()
	} else if integration {
		runningIntegration = true
		testutils.ConnectToTestDB()

		sourceDao = &dao.SourceDaoImpl{TenantID: &testutils.TestTenantData[0].Id}
		testutils.CreateFixtures()
	} else {
		sourceDao = &dao.MockSourceDao{}
	}

	code := t.Run()

	if integration {
		testutils.DropSchema()
	}

	os.Exit(code)
}
