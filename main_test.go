package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var (
	mockSourceDao          dao.SourceDao
	mockApplicationTypeDao dao.ApplicationTypeDao
	mockEndpointDao        dao.EndpointDao
	mockSourceTypeDao      dao.SourceTypeDao
	mockApplicationDao     dao.ApplicationDao
	mockMetaDataDao        dao.MetaDataDao

	flags parser.Flags
)

func TestMain(t *testing.M) {
	l.InitLogger(conf)

	flags = parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		database.ConnectAndMigrateDB("public")

		getSourceDao = getSourceDaoWithTenant
		getApplicationDao = getApplicationDaoWithTenant
		getEndpointDao = getEndpointDaoWithTenant
		getApplicationTypeDao = getApplicationTypeDaoWithTenant
		getSourceTypeDao = getSourceTypeDaoWithoutTenant
		getMetaDataDao = getMetaDataDaoWithTenant

		database.CreateFixtures()
		err := dao.PopulateStaticTypeCache()
		if err != nil {
			panic("failed to populate static type cache")
		}
	} else {
		mockSourceDao = &dao.MockSourceDao{Sources: fixtures.TestSourceData}
		mockApplicationDao = &dao.MockApplicationDao{Applications: fixtures.TestApplicationData}
		mockEndpointDao = &dao.MockEndpointDao{Endpoints: fixtures.TestEndpointData}
		mockSourceTypeDao = &dao.MockSourceTypeDao{SourceTypes: fixtures.TestSourceTypeData}
		mockApplicationTypeDao = &dao.MockApplicationTypeDao{ApplicationTypes: fixtures.TestApplicationTypeData}
		mockMetaDataDao = &dao.MockMetaDataDao{MetaDatas: fixtures.TestMetaDataData}

		getSourceDao = func(c echo.Context) (dao.SourceDao, error) { return mockSourceDao, nil }
		getApplicationDao = func(c echo.Context) (dao.ApplicationDao, error) { return mockApplicationDao, nil }
		getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) { return mockEndpointDao, nil }
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) { return mockSourceTypeDao, nil }
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) { return mockApplicationTypeDao, nil }

		getMetaDataDao = func(c echo.Context) (dao.MetaDataDao, error) { return mockMetaDataDao, nil }
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("public")
	}

	os.Exit(code)
}

func AssertLinks(t *testing.T, path string, links util.Links, limit int, offset int) {
	expectedFirstLink := fmt.Sprintf("%s?limit=%d&offset=%d", path, limit, offset)
	expectedLastLink := fmt.Sprintf("%s?limit=%d&offset=%d", path, limit, limit+offset)
	if links.First != expectedFirstLink {
		t.Error("first link is not correct for " + path)
	}

	if links.Last != expectedLastLink {
		t.Error("last link is not correct for " + path)
	}
}

func ErrorHandlingContext(handler echo.HandlerFunc) func(echo.Context) error {
	return middleware.HandleErrors(handler)
}
