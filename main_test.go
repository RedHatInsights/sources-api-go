package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var (
	e                      *echo.Echo
	mockSourceDao          dao.SourceDao
	mockApplicationTypeDao dao.ApplicationTypeDao
	mockEndpointDao        dao.EndpointDao
	mockSourceTypeDao      dao.SourceTypeDao
	mockApplicationDao     dao.ApplicationDao
	mockMetaDataDao        dao.MetaDataDao

	testDbName = "sources_api_test_go"
)

func TestMain(t *testing.M) {
	// flag to control running unit tests or connecting to a real db, usage:
	// go test -integration
	integration := flag.Bool("integration", false, "run unit or integration tests")
	createdb := flag.Bool("createdb", false, "create the test database")
	flag.Parse()
	l.InitLogger(conf)

	if *createdb {
		fmt.Fprintf(os.Stderr, "creating database %v...", testDbName)
		testutils.CreateTestDB()
	} else if *integration {
		testutils.ConnectToTestDB()
		getSourceDao = getSourceDaoWithTenant
		getApplicationDao = getApplicationDaoWithTenant
		getEndpointDao = getEndpointDaoWithTenant
		getApplicationTypeDao = getApplicationTypeDaoWithTenant
		getSourceTypeDao = getSourceTypeDaoWithoutTenant
		getMetaDataDao = getMetaDataDaoWithTenant

		dao.DB.Create(&m.Tenant{Id: 1})

		dao.DB.Create(testSourceTypeData)
		dao.DB.Create(testApplicationTypeData)

		dao.DB.Create(testSourceData)
		dao.DB.Create(testApplicationData)
		dao.DB.Create(testEndpointData)

		dao.DB.Create(testMetaData)
	} else {
		mockSourceDao = &dao.MockSourceDao{Sources: testSourceData}
		mockApplicationDao = &dao.MockApplicationDao{Applications: testApplicationData}
		mockEndpointDao = &dao.MockEndpointDao{Endpoints: testEndpointData}
		mockSourceTypeDao = &dao.MockSourceTypeDao{SourceTypes: testSourceTypeData}
		mockApplicationTypeDao = &dao.MockApplicationTypeDao{ApplicationTypes: testApplicationTypeData}
		mockMetaDataDao = &dao.MockMetaDataDao{MetaDatas: testMetaData}

		getSourceDao = func(c echo.Context) (dao.SourceDao, error) { return mockSourceDao, nil }
		getApplicationDao = func(c echo.Context) (dao.ApplicationDao, error) { return mockApplicationDao, nil }
		getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) { return mockEndpointDao, nil }
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) { return mockSourceTypeDao, nil }
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) { return mockApplicationTypeDao, nil }

		getMetaDataDao = func(c echo.Context) (dao.MetaDataDao, error) { return mockMetaDataDao, nil }

	}

	e = echo.New()
	code := t.Run()

	if *integration {
		testutils.DropSchema()
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
