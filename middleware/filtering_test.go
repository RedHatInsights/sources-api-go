package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
)

var e *echo.Echo

func TestMain(t *testing.M) {
	e = echo.New()
	code := t.Run()
	os.Exit(code)
}

func TestParseFilterWithOperation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v2.1/sources?filter[name][eq]=test", nil)
	c := e.NewContext(req, nil)

	parseFilterIntoRequest(c)

	filters, ok := c.Get("filters").([]Filter)
	if !ok {
		t.Error("filter did not parse correctly")
	}

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

	parseFilterIntoRequest(c)

	filters, ok := c.Get("filters").([]Filter)
	if !ok {
		t.Error("filter did not parse correctly")
	}

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
