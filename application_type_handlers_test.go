package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

var testApplicationTypeData = []m.ApplicationType{
	{Id: 1, DisplayName: "test app type"},
}

func TestSourceApplicationTypeSubcollectionList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources/:source_id/application_types", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 99)
	c.Set("offset", -1)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))
	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceListApplicationTypes(c)
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

	if out.Meta.Limit != 99 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != -1 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != 1 {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		if s["display_name"] != "test app type" {
			t.Error("ghosts infected the return")
		}
	}
}

func TestApplicationTypeList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/application_types", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("withoutTenancy", true)

	err := ApplicationTypeList(c)
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

	if len(out.Data) != 1 {
		t.Error("not enough objects passed back from DB")
	}

	for _, apptype := range out.Data {
		s, ok := apptype.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a application type response")
		}

		if s["display_name"] != "test app type" {
			t.Error("ghosts infected the return")
		}
	}
}

func TestApplicationTypeGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/application_types/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("withoutTenancy", true)

	err := ApplicationTypeGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outAppType m.ApplicationTypeResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outAppType)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if outAppType.DisplayName != "test app type" {
		t.Error("ghosts infected the return")
	}
}
