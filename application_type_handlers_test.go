package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceApplicationTypeSubcollectionList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
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

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationTypeList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types",
		nil,
		map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []util.Filter{},
		},
	)

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

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationTypeGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1",
		nil,
		map[string]interface{}{},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

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

func TestApplicationTypeGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/123",
		nil,
		map[string]interface{}{},
	)

	c.SetParamNames("id")
	c.SetParamValues("123")

	err := ApplicationTypeGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 404 {
		t.Error("Did not return 404")
	}
}
