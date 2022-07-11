package main

import (
	"os"
	"sort"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

var (
	mockSourceDao                    dao.SourceDao
	mockApplicationTypeDao           dao.ApplicationTypeDao
	mockEndpointDao                  dao.EndpointDao
	mockSourceTypeDao                dao.SourceTypeDao
	mockApplicationDao               dao.ApplicationDao
	mockMetaDataDao                  dao.MetaDataDao
	mockRhcConnectionDao             dao.RhcConnectionDao
	mockApplicationAuthenticationDao dao.ApplicationAuthenticationDao
	mockAuthenticationDao            dao.AuthenticationDao
)

func TestMain(t *testing.M) {
	l.InitLogger(conf)

	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		database.ConnectAndMigrateDB("public")

		getSourceDao = getSourceDaoWithTenant
		getApplicationDao = getApplicationDaoWithTenant
		getEndpointDao = getEndpointDaoWithTenant
		getApplicationTypeDao = getApplicationTypeDaoWithTenant
		getSourceTypeDao = getSourceTypeDaoWithoutTenant
		getMetaDataDao = getMetaDataDaoWithoutTenant
		getRhcConnectionDao = getDefaultRhcConnectionDao
		getApplicationAuthenticationDao = getApplicationAuthenticationDaoWithTenant
		getAuthenticationDao = getAuthenticationDaoWithTenant

		dao.Vault = &mocks.MockVault{}

		service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &mocks.MockSender{}} }

		database.CreateFixtures("public")

		err := dao.PopulateStaticTypeCache()
		if err != nil {
			panic("failed to populate static type cache")
		}
	} else {
		mockSourceDao = &dao.MockSourceDao{Sources: fixtures.TestSourceData, RelatedSources: fixtures.TestSourceData}
		mockApplicationDao = &dao.MockApplicationDao{Applications: fixtures.TestApplicationData}
		mockEndpointDao = &dao.MockEndpointDao{Endpoints: fixtures.TestEndpointData}
		mockSourceTypeDao = &dao.MockSourceTypeDao{SourceTypes: fixtures.TestSourceTypeData}
		mockApplicationTypeDao = &dao.MockApplicationTypeDao{ApplicationTypes: fixtures.TestApplicationTypeData}
		mockMetaDataDao = &dao.MockMetaDataDao{MetaDatas: fixtures.TestMetaDataData}
		mockRhcConnectionDao = &dao.MockRhcConnectionDao{RhcConnections: fixtures.TestRhcConnectionData, RelatedRhcConnections: fixtures.TestRhcConnectionData}
		mockApplicationAuthenticationDao = &dao.MockApplicationAuthenticationDao{ApplicationAuthentications: fixtures.TestApplicationAuthenticationData}
		mockAuthenticationDao = &dao.MockAuthenticationDao{Authentications: fixtures.TestAuthenticationData}

		getSourceDao = func(c echo.Context) (dao.SourceDao, error) { return mockSourceDao, nil }
		getApplicationDao = func(c echo.Context) (dao.ApplicationDao, error) { return mockApplicationDao, nil }
		getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) { return mockEndpointDao, nil }
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) { return mockSourceTypeDao, nil }
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) { return mockApplicationTypeDao, nil }
		getMetaDataDao = func(c echo.Context) (dao.MetaDataDao, error) { return mockMetaDataDao, nil }
		getRhcConnectionDao = func(c echo.Context) (dao.RhcConnectionDao, error) { return mockRhcConnectionDao, nil }
		getApplicationAuthenticationDao = func(c echo.Context) (dao.ApplicationAuthenticationDao, error) {
			return mockApplicationAuthenticationDao, nil
		}
		getAuthenticationDao = func(c echo.Context) (dao.AuthenticationDao, error) { return mockAuthenticationDao, nil }

	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("public")
	}

	os.Exit(code)
}

func SortByStringValueOnKey(field string, data []interface{}) {
	sort.SliceStable(data, func(i, j int) bool {
		dataI := data[i].(map[string]interface{})
		dataJ := data[j].(map[string]interface{})
		return dataI[field].(string) < dataJ[field].(string)
	})
}

func ErrorHandlingContext(handler echo.HandlerFunc) func(echo.Context) error {
	return middleware.HandleErrors(handler)
}
