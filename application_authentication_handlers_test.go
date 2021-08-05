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

var testApplicationAuthenticationData = []m.ApplicationAuthentication{
	{ID: 1, ApplicationID: 1, AuthenticationID: 1, TenantID: 1},
	{ID: 2, ApplicationID: 1, AuthenticationID: 2, TenantID: 1},
}

func TestApplicationAuthenticationList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/application_authentications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("limit", 100)
	c.Set("offset", 0)
	c.Set("filters", []middleware.Filter{})
	c.Set("tenantID", int64(1))

	err := ApplicationAuthenticationList(c)
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
			t.Error("model did not deserialize as a application")
		}
	}
}

func TestApplicationAuthenticationGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sources/v3.1/application_authentications/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("tenantID", int64(1))

	err := ApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outApplicationAuthentication m.ApplicationAuthenticationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outApplicationAuthentication)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}
}
