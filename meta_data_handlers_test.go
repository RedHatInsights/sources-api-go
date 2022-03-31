package main

import (
	"encoding/json"
	"fmt"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestApplicationTypeMetaDataSubcollectionList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/:application_type_id/app_meta_data",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("1")

	err := ApplicationTypeListMetaData(c)
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

	SortByStringValueOnKey("id", out.Data)

	m1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if m1["id"] != "1" {
		t.Error("ghosts infected the return")
	}

	m2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if m2["id"] != "2" {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationTypeMetaDataSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/3908503985/app_meta_data",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("3908503985")

	notFoundApplicationTypeListMetaData := ErrorHandlingContext(ApplicationTypeListMetaData)
	err := notFoundApplicationTypeListMetaData(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationTypeMetaDataSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/xxx/app_meta_data",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("xxx")

	badRequestApplicationTypeListMetaData := ErrorHandlingContext(ApplicationTypeListMetaData)
	err := badRequestApplicationTypeListMetaData(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationTypeMetaDataSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/app_meta_data",
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

	c.SetParamNames("application_type_id")
	c.SetParamValues("1")

	badRequestApplicationTypeListMetaData := ErrorHandlingContext(ApplicationTypeListMetaData)
	err := badRequestApplicationTypeListMetaData(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationTypeMetaDataSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	appTypeId := int64(1)

	// How many app meta data for application type id
	// is in fixtures
	var wantAppMetaDataCount int
	for _, appMetaData := range fixtures.TestMetaDataData {
		if appMetaData.ApplicationTypeID == appTypeId {
			wantAppMetaDataCount++
		}
	}

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/application_types/:application_type_id/app_meta_data",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		c.SetParamNames("application_type_id")
		c.SetParamValues(fmt.Sprintf("%d", appTypeId))

		err := ApplicationTypeListMetaData(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Error("Did not return 200")
		}

		var out util.Collection
		err = json.Unmarshal(rec.Body.Bytes(), &out)
		if err != nil {
			t.Error("Failed unmarshaling output")
		}

		if out.Meta.Limit != i["limit"] {
			t.Error("limit not set correctly")
		}

		if out.Meta.Offset != i["offset"] {
			t.Error("offset not set correctly")
		}

		if out.Meta.Count != wantAppMetaDataCount {
			t.Errorf("count not set correctly, got %d, want %d", out.Meta.Count, wantAppMetaDataCount)
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := wantAppMetaDataCount - i["offset"]
		if want < 0 {
			want = 0
		}

		if want > i["limit"] {
			want = i["limit"]
		}
		if got != want {
			t.Errorf("objects passed back from DB: want'%v', got '%v'", want, got)
		}

		AssertLinks(t, c.Request().RequestURI, out.Links, i["limit"], i["offset"])
	}
}

func TestMetaDataList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/app_meta_data",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	err := MetaDataList(c)
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

	if len(out.Data) != len(fixtures.TestMetaDataData) {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		_, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a application")
		}
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestMetaDataListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/app_meta_data",
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

	badRequestMetaDataList := ErrorHandlingContext(MetaDataList)
	err := badRequestMetaDataList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestMetaDataListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/app_meta_data",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		err := MetaDataList(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Error("Did not return 200")
		}

		var out util.Collection
		err = json.Unmarshal(rec.Body.Bytes(), &out)
		if err != nil {
			t.Error("Failed unmarshaling output")
		}

		if out.Meta.Limit != i["limit"] {
			t.Error("limit not set correctly")
		}

		if out.Meta.Offset != i["offset"] {
			t.Error("offset not set correctly")
		}

		if out.Meta.Count != len(fixtures.TestMetaDataData) {
			t.Errorf("count not set correctly")
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := len(fixtures.TestMetaDataData) - i["offset"]
		if want < 0 {
			want = 0
		}

		if want > i["limit"] {
			want = i["limit"]
		}
		if got != want {
			t.Errorf("objects passed back from DB: want'%v', got '%v'", want, got)
		}

		AssertLinks(t, c.Request().RequestURI, out.Links, i["limit"], i["offset"])
	}
}

func TestMetaDataGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/app_meta_data/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := MetaDataGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outMetaData m.MetaDataResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outMetaData)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}
}

func TestMetaDataGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/app_meta_data/13984739874",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("13984739874")

	notFoundMetaDataGet := ErrorHandlingContext(MetaDataGet)
	err := notFoundMetaDataGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestMetaDataGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/app_meta_data/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestMetaDataGet := ErrorHandlingContext(MetaDataGet)
	err := badRequestMetaDataGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}
