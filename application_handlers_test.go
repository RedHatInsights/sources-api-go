package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

var testApplicationData = []m.Application{
	{ID: 1, Extra: datatypes.JSON(getExtraValue("{\"extra\": true}")), ApplicationTypeID: 1, SourceID: 1, TenantID: 1},
	{ID: 2, Extra: datatypes.JSON(getExtraValue("{\"extra\": false}")), ApplicationTypeID: 1, SourceID: 1, TenantID: 1},
}

func getExtraValue(val string) json.RawMessage {
	var out json.RawMessage

	err := json.Unmarshal([]byte(val), &out)
	if err != nil {
		panic(err)
	}

	return out
}

func TestSourceApplicationSubcollectionList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources/1/applications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))
	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceListApplications(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != 2 {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}

		if s["id"] != "1" && s["id"] != "2" {
			t.Error("ghosts infected the return")
		}
	}

	AssertLinks(t, req.RequestURI, out.Links, 100, 0)
}

func TestApplicationList(t *testing.T) {
	path := "/api/sources/v3.1/applications"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))

	err := ApplicationList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != 2 {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a application")
		}

		if s["extra"] == nil {
			t.Error("ghosts infected the return")
		}
	}

	AssertLinks(t, req.RequestURI, out.Links, 100, 0)
}

func TestApplicationGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/applications/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("tenantID", int64(1))

	err := ApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outApplication m.ApplicationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outApplication)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if outApplication.Extra == nil {
		t.Error("ghosts infected the return")
	}
}
