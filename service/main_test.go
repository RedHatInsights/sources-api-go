package service

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/logger"
)

var (
	endpointDao dao.EndpointDao
	sourceDao   dao.SourceDao
)

func TestMain(t *testing.M) {
	logger.InitLogger(config.Get())

	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		database.ConnectAndMigrateDB("service")

		endpointDao = dao.GetEndpointDao(&fixtures.TestTenantData[0].Id)
		sourceDao = dao.GetSourceDao(&dao.RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
		database.CreateFixtures("service")
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
