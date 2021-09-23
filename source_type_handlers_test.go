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

var testSourceTypeData = []m.SourceType{
	{Id: 1, Name: "amazon"},
}

func TestSourceTypeList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/source_types", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})

	err := SourceTypeList(c)

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

	for _, sourceType := range out.Data {
		s, ok := sourceType.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a application type response")
		}

		if s["name"] != "amazon" {
			t.Error("ghosts infected the return")
		}
	}

	AssertLinks(t, req.RequestURI, out.Links, 100, 0)
}
