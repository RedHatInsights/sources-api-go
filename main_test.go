package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	e                      *echo.Echo
	mockSourceDao          dao.SourceDao
	mockApplicationTypeDao dao.ApplicationTypeDao
	mockEndpointDao        dao.EndpointDao
	mockSourceTypeDao      dao.SourceTypeDao
	mockApplicationDao     dao.ApplicationDao

	testDbName = "sources_api_test_go"
)

func TestMain(t *testing.M) {
	// flag to control running unit tests or connecting to a real db, usage:
	// go test -integration
	integration := flag.Bool("integration", false, "run unit or integration tests")
	createdb := flag.Bool("createdb", false, "create the test database")
	flag.Parse()

	if *createdb {
		fmt.Fprintf(os.Stderr, "creating database %v...", testDbName)
		err := createTestDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating test DB: %v", err)
			os.Exit(1)
		}

		os.Exit(0)
	} else if *integration {
		connectToTestDB()
		getSourceDao = getSourceDaoWithTenant
		getApplicationDao = getApplicationDaoWithTenant
		getEndpointDao = getEndpointDaoWithTenant
		getApplicationTypeDao = getApplicationTypeDaoWithoutTenant
		getSourceTypeDao = getSourceTypeDaoWithoutTenant

		dao.DB.Create(&m.Tenant{Id: 1})

		dao.DB.Create(testSourceTypeData)
		dao.DB.Create(testApplicationTypeData)

		dao.DB.Create(testSourceData)
		dao.DB.Create(testApplicationData)
		dao.DB.Create(testEndpointData)
	} else {
		mockSourceDao = &dao.MockSourceDao{Sources: testSourceData}
		mockApplicationDao = &dao.MockApplicationDao{Applications: testApplicationData}
		mockEndpointDao = &dao.MockEndpointDao{Endpoints: testEndpointData}
		mockSourceTypeDao = &dao.MockSourceTypeDao{SourceTypes: testSourceTypeData}
		mockApplicationTypeDao = &dao.MockApplicationTypeDao{ApplicationTypes: testApplicationTypeData}

		getSourceDao = func(c echo.Context) (dao.SourceDao, error) { return mockSourceDao, nil }
		getApplicationDao = func(c echo.Context) (dao.ApplicationDao, error) { return mockApplicationDao, nil }
		getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) { return mockEndpointDao, nil }
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) { return mockSourceTypeDao, nil }
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) { return mockApplicationTypeDao, nil }
	}

	e = echo.New()
	code := t.Run()

	if *integration {
		dao.DB.Exec("DROP TABLE applications")
		dao.DB.Exec("DROP TABLE endpoints")
		dao.DB.Exec("DROP TABLE sources")
		dao.DB.Exec("DROP TABLE application_types")
		dao.DB.Exec("DROP TABLE source_types")
		dao.DB.Exec("DROP TABLE tenants")
	}

	os.Exit(code)
}

func connectToTestDB() {
	db, err := gorm.Open(postgres.Open(testDbString(testDbName)), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "db must not exist - create the database '%v' first with '-createdb'.", testDbName)
		panic(err)
	}
	dao.DB = db
	rawDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	rawDB.SetMaxOpenConns(20)

	// migrate all of the models.
	err = db.AutoMigrate(
		&m.SourceType{},
		&m.ApplicationType{},

		&m.Source{},
		&m.Application{},
		&m.Endpoint{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error automigrating the schema: %v", err)
		os.Exit(1)
	}
}

func createTestDB() error {
	db, err := gorm.Open(postgres.Open(testDbString("postgres")), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	out := db.Exec(fmt.Sprintf("CREATE DATABASE %v", testDbName))
	return out.Error
}

func testDbString(dbname string) string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		dbname,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
	)
}
