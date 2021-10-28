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

	endpointDao dao.EndpointDao
	sourceDao   dao.SourceDao
)

func TestMain(t *testing.M) {
	flags := testutils.ParseFlags()

	if flags.CreateDb {
		testutils.CreateTestDB()
	} else if flags.Integration {
		runningIntegration = true
		testutils.ConnectToTestDB()

		endpointDao = &dao.EndpointDaoImpl{TenantID: &testutils.TestTenantData[0].Id}
		sourceDao = &dao.SourceDaoImpl{TenantID: &testutils.TestTenantData[0].Id}
		testutils.CreateFixtures()
	} else {
		endpointDao = &dao.MockEndpointDao{}
		sourceDao = &dao.MockSourceDao{}
	}

	code := t.Run()

	if flags.Integration {
		testutils.DropSchema()
	}

	os.Exit(code)
}
