package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceApplicationSubcollectionList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/applications",
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

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceApplicationSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/134793847/applications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("134793847")

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err := notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

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

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

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

func TestApplicationGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications/9843762095",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9843762095")

	notFoundApplicationGet := ErrorHandlingContext(ApplicationGet)
	err := notFoundApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationCreateGood(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		SourceIDRaw:          "2",
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 201 {
		t.Errorf("Wrong return code, expected %v got %v", 201, rec.Code)
	}

	app := m.ApplicationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &app)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if app.SourceID != "2" {
		t.Errorf("Wrong source ID, wanted %v got %v", "2", app.SourceID)
	}

	id, _ := strconv.ParseInt(app.ID, 10, 64)
	dao, _ := getApplicationDao(c)
	_ = dao.Delete(&id)
}

func TestApplicationCreateMissingSourceId(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 400 {
		t.Errorf("Wrong return code, expected %v got %v", 400, rec.Code)
	}
}

func TestApplicationCreateMissingApplicationTypeId(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		SourceIDRaw: "1",
		Extra:       nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 400 {
		t.Errorf("Wrong return code, expected %v got %v", 400, rec.Code)
	}
}

func TestApplicationCreateIncompatible(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: false}

	req := m.ApplicationCreateRequest{
		SourceIDRaw:          "2",
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 400 {
		t.Errorf("Wrong return code, expected %v got %v", 400, rec.Code)
	}
}
func TestApplicationEdit(t *testing.T) {
	req := m.ApplicationEditRequest{
		Extra:                   []byte(`{"thing": true}`),
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/applications/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationEdit(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Errorf("Wrong return code, expected %v got %v", 200, rec.Code)
	}

	app := m.ApplicationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &app)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if app.AvailabilityStatus.AvailabilityStatus != "available" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", app.AvailabilityStatus.AvailabilityStatus)
	}
}
