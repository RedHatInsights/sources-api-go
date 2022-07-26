package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceApplicationTypeSubcollectionList(t *testing.T) {
	tenantId := int64(1)
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

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

	// We are looking for source's applications and then application
	// types of these applications
	appTypes := make(map[int64]int)
	for _, app := range fixtures.TestApplicationData {
		if app.SourceID == sourceId {
			appTypes[app.ApplicationTypeID]++
		}
	}

	if len(out.Data) != len(appTypes) {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		if s["id"] == "1" && s["display_name"] != "test app type" {
			t.Error("ghosts infected the return")
		}
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestSourceApplicationTypeSubcollectionListEmptyList tests that empty list is
// returned for a source without related applications
func TestSourceApplicationTypeSubcollectionListEmptyList(t *testing.T) {
	tenantId := int64(1)
	sourceId := int64(101)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	err := SourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestSourceApplicationTypeSubcollectionListTenantNotExist tests that not found is returned
// for not existing tenant
func TestSourceApplicationTypeSubcollectionListTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceListApplicationTypes := ErrorHandlingContext(SourceListApplicationTypes)
	err := notFoundSourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestSourceApplicationTypeSubcollectionListInvalidTenant tests that not found is returned
// for tenant who doesn't own the source
func TestSourceApplicationTypeSubcollectionListInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceListApplicationTypes := ErrorHandlingContext(SourceListApplicationTypes)
	err := notFoundSourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceApplicationTypeSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/109830938/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("109830938")

	notFoundSourceListApplicationTypes := ErrorHandlingContext(SourceListApplicationTypes)
	err := notFoundSourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceApplicationTypeSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/application_types",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("xxx")

	badRequestSourceListApplicationTypes := ErrorHandlingContext(SourceListApplicationTypes)
	err := badRequestSourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceApplicationTypeSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/application_types",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	badRequestSourceListApplicationTypes := ErrorHandlingContext(SourceListApplicationTypes)
	err := badRequestSourceListApplicationTypes(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
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

	if len(out.Data) != len(fixtures.TestApplicationTypeData) {
		t.Error("not enough objects passed back from DB")
	}

	for _, appType := range out.Data {
		s, ok := appType.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a application type response")
		}

		if s["id"] == "1" && s["display_name"] != "test app type" {
			t.Error("ghosts infected the return")
		}
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestApplicationTypeListWithTenant tests that list of application types is returned
// even when tenant is provided (the request usually doesn't need a tenant)
func TestApplicationTypeListWithTenant(t *testing.T) {
	tenantId := int64(1)
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types",
		nil,
		map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []util.Filter{},
			"tenant":  tenantId,
		},
	)

	err := ApplicationTypeList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}
}

func TestApplicationTypeListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
		},
	)

	badRequestApplicationTypeList := ErrorHandlingContext(ApplicationTypeList)
	err := badRequestApplicationTypeList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationTypeGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1",
		nil,
		nil,
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

// TestApplicationTypeGetWithTenant tests that application type is returned
// even when tenant is provided (the request usually doesn't need a tenant)
func TestApplicationTypeGetWithTenant(t *testing.T) {
	tenantId := int64(1)
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1",
		nil,
		map[string]interface{}{
			"tenant": tenantId,
		},
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
		"/api/sources/v3.1/application_types/12362095",
		nil,
		nil,
	)

	c.SetParamNames("id")
	c.SetParamValues("12362095")

	notFoundApplicationTypeGet := ErrorHandlingContext(ApplicationTypeGet)
	err := notFoundApplicationTypeGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationTypeGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/xxx",
		nil,
		nil,
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationTypeGet := ErrorHandlingContext(ApplicationTypeGet)
	err := badRequestApplicationTypeGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}
