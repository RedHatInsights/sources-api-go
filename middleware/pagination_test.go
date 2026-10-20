package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestParsePagination(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources?limit=99&offset=50", nil)
	c := e.NewContext(req, nil)

	err := parsePaginationIntoContext(c)
	if err != nil {
		t.Error("something went very wrong - error parsing pagination.")
	}

	limit, ok := c.Get("limit").(int)
	if !ok {
		t.Error("limit did not get parsed correctly")
	}

	if limit != 99 {
		t.Error("limit not parsed to integer correctly")
	}

	offset, ok := c.Get("offset").(int)
	if !ok {
		t.Error("offset did not get parsed correctly")
	}

	if offset != 50 {
		t.Error("offset not parsed to integer correctly")
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources", nil)
	c := e.NewContext(req, nil)

	err := parsePaginationIntoContext(c)
	if err != nil {
		t.Error("something went very wrong - error parsing pagination.")
	}

	limit, ok := c.Get("limit").(int)
	if !ok {
		t.Error("limit did not get parsed correctly")
	}

	if limit != 100 {
		t.Error("limit not parsed to integer correctly")
	}

	offset, ok := c.Get("offset").(int)
	if !ok {
		t.Error("offset did not get parsed correctly")
	}

	if offset != 0 {
		t.Error("offset not parsed to integer correctly")
	}
}

func TestParsePaginationBadRequestInvalidLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources?limit=zzzz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	badRequestParsePaginationIntoContext := HandleErrors(parsePaginationIntoContext)

	err := badRequestParsePaginationIntoContext(c)
	if err != nil {
		t.Error("something went very wrong - error parsing pagination.")
	}

	templates.BadRequestTest(t, rec)

	var resp util.ErrorDocument

	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Error("Error unmarshaling response, malformed error document")
	}

	if !strings.Contains(resp.Errors[0].Detail, "error parsing") {
		t.Error("Error document not formed correctly")
	}
}

func TestParsePaginationBadRequestInvalidOffset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources?offset=zzzz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	badRequestParsePaginationIntoContext := HandleErrors(parsePaginationIntoContext)

	err := badRequestParsePaginationIntoContext(c)
	if err != nil {
		t.Error("something went very wrong - error parsing pagination.")
	}

	templates.BadRequestTest(t, rec)

	var resp util.ErrorDocument

	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Error("Error unmarshaling response, malformed error document")
	}

	if !strings.Contains(resp.Errors[0].Detail, "error parsing") {
		t.Error("Error document not formed correctly")
	}
}
