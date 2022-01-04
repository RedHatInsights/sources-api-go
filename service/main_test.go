package service

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
)

var (
	// runningIntegration is used to skip integration tests if we're just running unit tests.
	runningIntegration = false

	endpointDao dao.EndpointDao
	sourceDao   dao.SourceDao
)

func TestMain(t *testing.M) {
	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		runningIntegration = true
		database.ConnectAndMigrateDB("service")

		endpointDao = &dao.EndpointDaoImpl{TenantID: &fixtures.TestTenantData[0].Id}
		sourceDao = &dao.SourceDaoImpl{TenantID: &fixtures.TestTenantData[0].Id}
		database.CreateFixtures()
	} else {
		endpointDao = &dao.MockEndpointDao{}
		sourceDao = &dao.MockSourceDao{}
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("service")
	}

	os.Exit(code)
}
