package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	l "github.com/RedHatInsights/sources-api-go/logger"
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
)

func TestMain(t *testing.M) {
	l.InitLogger(conf)

	createDb, integration := testutils.ParseFlags()

	if createDb {
		testutils.CreateTestDB()
	} else if integration {
		testutils.ConnectToTestDB()
		getSourceDao = getSourceDaoWithTenant
		getApplicationDao = getApplicationDaoWithTenant
		getEndpointDao = getEndpointDaoWithTenant
		getApplicationTypeDao = getApplicationTypeDaoWithTenant
		getSourceTypeDao = getSourceTypeDaoWithoutTenant
		getMetaDataDao = getMetaDataDaoWithTenant

		testutils.CreateFixtures()
	} else {
		mockSourceDao = &dao.MockSourceDao{Sources: testutils.TestSourceData}
		mockApplicationDao = &dao.MockApplicationDao{Applications: testutils.TestApplicationData}
		mockEndpointDao = &dao.MockEndpointDao{Endpoints: testutils.TestEndpointData}
		mockSourceTypeDao = &dao.MockSourceTypeDao{SourceTypes: testutils.TestSourceTypeData}
		mockApplicationTypeDao = &dao.MockApplicationTypeDao{ApplicationTypes: testutils.TestApplicationTypeData}
		mockMetaDataDao = &dao.MockMetaDataDao{MetaDatas: testutils.TestMetaDataData}

		getSourceDao = func(c echo.Context) (dao.SourceDao, error) { return mockSourceDao, nil }
		getApplicationDao = func(c echo.Context) (dao.ApplicationDao, error) { return mockApplicationDao, nil }
		getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) { return mockEndpointDao, nil }
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) { return mockSourceTypeDao, nil }
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) { return mockApplicationTypeDao, nil }

		getMetaDataDao = func(c echo.Context) (dao.MetaDataDao, error) { return mockMetaDataDao, nil }

	}

	code := t.Run()

	if integration {
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
