package service

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/logger"
)

var (
	endpointDao   dao.EndpointDao
	sourceDao     dao.SourceDao
	requestParams *dao.RequestParams
)

func TestMain(t *testing.M) {
	logger.InitLogger(config.Get())

	flags := parser.ParseFlags()

	tenantId := fixtures.TestTenantData[0].Id
	requestParams = &dao.RequestParams{TenantID: &tenantId}

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		database.ConnectAndMigrateDB("service")

		endpointDao = dao.GetEndpointDao(&tenantId)
		sourceDao = dao.GetSourceDao(requestParams)

		database.CreateFixtures("service")
		err := dao.PopulateStaticTypeCache()
		if err != nil {
			panic("failed to populate static type cache")
		}
	} else {
		endpointDao = &mocks.MockEndpointDao{Endpoints: fixtures.TestEndpointData}
		sourceDao = &mocks.MockSourceDao{Sources: fixtures.TestSourceData}

		err := dao.PopulateMockStaticTypeCache()
		if err != nil {
			panic("failed to populate mock static type cache")
		}
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("service")
	}

	os.Exit(code)
}
