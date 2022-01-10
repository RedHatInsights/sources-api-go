package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/labstack/echo/v4"
)

var e *echo.Echo
var conf = config.Get()

func TestParseFilterWithOperation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v2.1/sources?filter[name][eq]=test", nil)
	c := e.NewContext(req, nil)

	filters := parseFilter(c)

	if len(filters) != 1 {
		t.Error("wrong number of filters")
	}

	f := filters[0]

	if f.Name != "name" {
		t.Error("did not parse field name correctly")
	}

	if f.Operation != "[eq]" {
		t.Error("did not parse operation correctly")
	}

	if f.Value[0] != "test" {
		t.Error("did not parse value correctly")
	}
}

func TestParseFilterWithoutOperation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v2.1/sources?filter[name]=test", nil)
	c := e.NewContext(req, nil)

	filters := parseFilter(c)

	if len(filters) != 1 {
		t.Error("wrong number of filters")
	}

	f := filters[0]

	if f.Name != "name" {
		t.Error("did not parse field name correctly")
	}

	if f.Operation != "" {
		t.Error("did not parse operation correctly")
	}

	if f.Value[0] != "test" {
		t.Error("did not parse value correctly")
	}
}

func TestParseSorting(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v2.1/sources?sort_by=name", nil)
	c := e.NewContext(req, nil)

	sortFilter := parseSorting(c)

	if sortFilter == nil {
		t.Error("sorting did not parse correctly")
		t.FailNow()
	}

	if len(sortFilter.Value) != 1 {
		t.Error("wrong number of sorts")
	}

	s := sortFilter.Value[0]

	if s != "name" {
		t.Error("sort value did not get parsed correctly")
	}
}

func TestParseSortingMultiple(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v2.1/sources?sort_by=name&sort_by=uid", nil)
	c := e.NewContext(req, nil)

	sortFilter := parseSorting(c)

	if sortFilter == nil {
		t.Error("sorting did not parse correctly")
		t.FailNow()
	}

	if len(sortFilter.Value) != 2 {
		t.Error("wrong number of sorts")
	}

	if sortFilter.Value[0] != "name" {
		t.Error("sort[0] value did not get parsed correctly")
	}

	if sortFilter.Value[1] != "uid" {
		t.Error("sort[1] value did not get parsed correctly")
	}
}
