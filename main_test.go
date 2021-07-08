package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
)

var (
	e                      *echo.Echo
	mockSourceDao          dao.SourceDao
	mockApplicationTypeDao dao.ApplicationTypeDao
	mockSourceTypeDao      dao.SourceTypeDao
)

func TestMain(t *testing.M) {
	// flag to control running unit tests or connecting to a real db, usage:
	// go test -integration
	integration := flag.Bool("integration", false, "run unit or integration tests")
	flag.Parse()

	if *integration {
		fmt.Fprintf(os.Stderr, "Integration not working yet, exiting.")
		os.Exit(1)
	} else {
		mockSourceDao = setupSourceMockDao(sources)
		mockApplicationTypeDao = setupApplicationTypeMockDao(apptypes)
		getApplicationTypeDao = func(c echo.Context) (dao.ApplicationTypeDao, error) {
			return mockApplicationTypeDao, nil
		}
		getSourceDao = func(c echo.Context) (dao.SourceDao, error) {
			return mockSourceDao, nil
		}

		mockSourceTypeDao = setupSourceTypeMockDao(sourceTypesData)
		getSourceTypeDao = func(c echo.Context) (dao.SourceTypeDao, error) {
			return mockSourceTypeDao, nil
		}
	}

	e = echo.New()
	code := t.Run()
	os.Exit(code)
}

func setupSourceMockDao(sources string) dao.SourceDao {
	a := make([]m.Source, 0, 10)
	err := json.Unmarshal([]byte(sources), &a)
	if err != nil {
		panic(err)
	}
	return &dao.MockSourceDao{Sources: a}
}

func setupApplicationTypeMockDao(apptypes string) dao.ApplicationTypeDao {
	a := make([]m.ApplicationType, 0, 10)
	err := json.Unmarshal([]byte(apptypes), &a)
	if err != nil {
		panic(err)
	}
	return &dao.MockApplicationTypeDao{ApplicationTypes: a}
}

func setupSourceTypeMockDao(sourceTypes string) dao.SourceTypeDao {
	a := make([]m.SourceType, 0, 10)
	err := json.Unmarshal([]byte(sourceTypes), &a)
	if err != nil {
		panic(err)
	}
	return &dao.MockSourceTypeDao{SourceTypes: a}
}
