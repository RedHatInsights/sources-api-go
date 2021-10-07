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

var testEndpointData = []m.Endpoint{
	{ID: 1, SourceID: 1, TenantID: 1},
	{ID: 2, SourceID: 1, TenantID: 1},
}

func TestSourceEndpointSubcollectionList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/sources/1/endpoints", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))
	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceListEndpoint(c)
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

func TestEndpointList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/endpoints", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))

	err := EndpointList(c)
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
		_, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a endpoint")
		}
	}

	AssertLinks(t, req.RequestURI, out.Links, 100, 0)
}

func TestEndpointGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/endpoints/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("tenantID", int64(1))

	err := EndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outEndpoint m.EndpointResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outEndpoint)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}
}

func TestEndpointGetNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/endpoints/970283452983", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("970283452983")
	c.Set("tenantID", int64(1))

	err := EndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 404 {
		t.Error("Did not return 404")
	}
}
