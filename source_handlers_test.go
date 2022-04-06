package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/helpers"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/kafka"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func TestSourceListAuthentications(t *testing.T) {
	tenantId := fixtures.TestTenantData[0].Id

	// If we're running integration tests without Vault...
	if parser.RunningIntegrationTests && !config.IsVaultOn() {
		// Create one authentication for the database tests, to make sure that we at least have one we can fetch.
		authsDao := dao.GetAuthenticationDao(&tenantId)
		err := authsDao.Create(&fixtures.TestAuthenticationData[0])
		if err != nil {
			t.Errorf(`could not create the authentication fixture for the test`)
		}
	} else {
		// If we're either running unit tests, or integration tests with Vault, we force the secret store to be "vault"
		// since there are multiple places where this "if config.IsVaultOn()" check is run.
		conf.SecretStore = "vault"
	}

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceListAuthentications(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	auth1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if config.IsVaultOn() {
		want := fixtures.TestAuthenticationData[0].ID
		got := auth1["id"]

		if want != got {
			t.Errorf(`the IDs from the authentication don't match. Want "%s", got "%s"'`, want, got)
		}
	} else {
		want := fixtures.TestAuthenticationData[0].DbID

		outputId, ok := auth1["id"].(string)
		if !ok {
			t.Errorf(`invalid ID received from authentication. Want "%s", got "%s"`, "string", reflect.TypeOf(auth1["id"]))
		}

		got, err := strconv.ParseInt(outputId, 10, 64)
		if err != nil {
			t.Errorf(`could not parse ID from authentication: %s`, err)
		}

		if want != got {
			t.Errorf(`the IDs from the authentication don't match. Want "%d", got "%d"'`, want, got)
		}
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListAuthenticationsNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/30983098439/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("30983098439")

	notFoundSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := notFoundSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceListAuthenticationsBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/authentications",
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

	badRequestSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := badRequestSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourceTypeSourceSubcollectionList(t *testing.T) {
	sourceTypeId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

	err := SourceTypeListSource(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	// How many sources with given source type is in fixtures
	// (adding new fixtures will not affect the test)
	var wantSourcesCount int
	for _, i := range fixtures.TestSourceData {
		if i.SourceTypeID == sourceTypeId {
			wantSourcesCount++
		}
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		_, ok := src.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}

	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// Existing source type + not existing source with this source type
// expected is Status OK + empty subcollection in response
func TestSourceTypeSourceSubcollectionListEmptySubcollection(t *testing.T) {
	sourceTypeId := int64(100)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

	err := SourceTypeListSource(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	// How many sources with given source type is in fixtures
	// (adding new fixtures will not affect the test)
	var wantSourcesCount int
	for _, i := range fixtures.TestSourceData {
		if i.SourceTypeID == sourceTypeId {
			wantSourcesCount++
		}
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		_, ok := src.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}

	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceTypeSourceSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/80398409384/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues("80398409384")

	notFoundSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := notFoundSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceTypeSourceSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/xxx/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues("xxx")

	badRequestSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := badRequestSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourceTypeSourceSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
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

	c.SetParamNames("source_type_id")
	c.SetParamValues("1")

	badRequestSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := badRequestSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourceTypeSourceSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	// How many sources with given source type is in fixtures
	// (adding new fixtures will not affect the test)
	sourceTypeId := fixtures.TestSourceTypeData[0].Id
	var wantSourcesCount int
	for _, i := range fixtures.TestSourceData {
		if i.SourceTypeID == sourceTypeId {
			wantSourcesCount++
		}
	}

	for _, i := range testData {

		c, rec := request.CreateTestContext(
			http.MethodGet,
			fmt.Sprintf("/api/sources/v3.1/source_types/%d/sources", sourceTypeId),
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		c.SetParamNames("source_type_id")
		c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

		err := SourceTypeListSource(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("want %d, got %d", http.StatusOK, rec.Code)
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

		if out.Meta.Count != wantSourcesCount {
			t.Errorf("count not set correctly")
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := wantSourcesCount - i["offset"]
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

func TestApplicatioTypeListSourceSubcollectionList(t *testing.T) {
	appTypeId := int64(1)
	wantSourcesCount := helpers.GetSourcesCountWithAppType(appTypeId)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues(fmt.Sprintf("%d", appTypeId))

	err := ApplicationTypeListSource(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		if s["id"] == 1 && s["name"] != "Source1" {
			t.Error("ghosts infected the return")
		}
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicatioTypeListSourceSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/398748974/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("398748974")

	notFoundApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := notFoundApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicatioTypeListSourceSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/xxx/sources",
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

	badRequestApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := badRequestApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicatioTypeListSourceSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
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

	badRequestApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := badRequestApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationTypeListSourceSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	// How many sources with given application type is in fixtures
	// => check applications with given app type and for matched apps
	// find sources
	// (adding new fixtures will not affect the test)
	appTypeId := int64(1)
	wantSourcesCount := helpers.GetSourcesCountWithAppType(appTypeId)

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			fmt.Sprintf("/api/sources/v3.1/application_types/%d/sources", appTypeId),
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

		err := ApplicationTypeListSource(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("want %d, got %d", http.StatusOK, rec.Code)
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

		if out.Meta.Count != wantSourcesCount {
			t.Errorf("count not set correctly, got %d, want %d", out.Meta.Count, wantSourcesCount)
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := wantSourcesCount - i["offset"]
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

func TestSourceList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		})

	err := SourceList(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if out.Meta.Count != len(fixtures.TestSourceData) {
		t.Errorf("count not set correctly")
	}

	if len(out.Data) != len(fixtures.TestSourceData) {
		t.Error("not enough objects passed back from DB")
	}

	SortByStringValueOnKey("name", out.Data)

	s1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if s1["name"] != "Source1" {
		t.Error("ghosts infected the return")
	}

	s2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if s2["name"] != "Source2" {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListWithOffsetAndLimit(t *testing.T) {
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
			"/api/sources/v3.1/sources",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			})

		err := SourceList(c)
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

		if out.Meta.Count != len(fixtures.TestSourceData) {
			t.Errorf("count not set correctly")
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := len(fixtures.TestSourceData) - i["offset"]
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

func TestSourceListSatellite(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
			// this gets set during the parse middleware
			"cert-auth": true,
		})

	err := SourceList(c)
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

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != 0 {
		t.Error("Objects were not filtered out of request")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		})
	badRequestSourceList := ErrorHandlingContext(SourceList)
	err := badRequestSourceList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourceGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := SourceGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Error("Did not return 200")
	}

	var outSrc m.SourceResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outSrc)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if *outSrc.Name != "Source1" {
		t.Error("ghosts infected the return")
	}
}

func TestSourceGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/9872034520975",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9872034520975")

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err := notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestSourceGet := ErrorHandlingContext(SourceGet)
	err := badRequestSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

// TestSourceCreateBadRequest tests that the handler responds with an 400 when an invalid JSON is received
func TestSourceCreateBadRequest(t *testing.T) {
	emptyName := ""
	requestBody := m.SourceCreateRequest{
		Name: &emptyName,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestSourceCreate := ErrorHandlingContext(SourceCreate)
	err = badRequestSourceCreate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

// TestSourceCreate tests that a 201 is received when a proper JSON message is received
func TestSourceCreate(t *testing.T) {
	// Test with a proper JSON
	name := "TestRequest"
	uid := "5"
	version := "10.5"
	imported := "true"
	sourceRef := "Source reference #5"
	var sourceTypeId int64 = 1

	requestBody := m.SourceCreateRequest{
		Name:                &name,
		Uid:                 &uid,
		Version:             &version,
		Imported:            &imported,
		SourceRef:           &sourceRef,
		AppCreationWorkflow: m.AccountAuth,
		AvailabilityStatus:  m.Available,
		SourceTypeIDRaw:     &sourceTypeId,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = SourceCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Did not return 201. Body: %s", rec.Body.String())
	}

	src := m.SourceResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &src)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if src.SourceTypeId != "1" {
		t.Errorf("Wrong source ID, wanted %v got %v", "1", src.SourceTypeId)
	}

	id, _ := strconv.ParseInt(src.ID, 10, 64)
	dao, _ := getSourceDao(c)
	_, err = dao.Delete(&id)
	if err != nil {
		t.Errorf("Failed to delete source id %v", id)
	}
}

func TestSourceEdit(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               request.PointerToString(newSourceName),
		AvailabilityStatus: request.PointerToString("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := SourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusOK, rec.Code)
	}

	src := m.SourceResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &src)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if *src.Name != newSourceName {
		t.Errorf("Unexpected source name: expected '%s', got '%s'", newSourceName, *src.Name)
	}

	if *src.AvailabilityStatus != "available" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", *src.AvailabilityStatus)
	}
}

func TestSourceEditNotFound(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               request.PointerToString(newSourceName),
		AvailabilityStatus: request.PointerToString("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/8937498374",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("8937498374")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundSourceEdit := ErrorHandlingContext(SourceEdit)
	err := notFoundSourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceEditBadRequest(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               request.PointerToString(newSourceName),
		AvailabilityStatus: request.PointerToString("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/xxx",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestSourceEdit := ErrorHandlingContext(SourceEdit)
	err := badRequestSourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourceDelete(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/100",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("100")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := SourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusNoContent, rec.Code)
	}

	// Check that source doesn't exist.
	c, rec = request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/100",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("100")

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err = notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/9038049384",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9038049384")

	notFoundSourceDelete := ErrorHandlingContext(SourceDelete)
	err := notFoundSourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceDeleteBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestSourceDelete := ErrorHandlingContext(SourceDelete)
	err := badRequestSourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestAvailabilityStatusCheck(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 202 {
		t.Errorf("Wrong code, got %v, expected %v", rec.Code, 202)
	}
}

func TestAvailabilityStatusCheckNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/183209745/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("183209745")

	notFoundSourceCheckAvailability := ErrorHandlingContext(SourceCheckAvailability)
	err := notFoundSourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestAvailabilityStatusCheckBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/xxx/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("xxx")

	badRequestSourceCheckAvailability := ErrorHandlingContext(SourceCheckAvailability)
	err := badRequestSourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTest(t *testing.T) {
	sourceId := "1"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/"+sourceId+"/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(sourceId)

	err := SourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("want %d, got %d", http.StatusOK, rec.Code)
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshalling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if parser.RunningIntegrationTests {
		if len(out.Data) != 2 {
			t.Error("not enough objects passed back from DB")
		}
	} else {
		if len(fixtures.TestRhcConnectionData) != len(out.Data) {
			t.Error("not enough objects passed back from DB")
		}
	}

	for _, source := range out.Data {
		_, ok := source.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}
	}
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/0394830498/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("0394830498")

	notFoundSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := notFoundSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/rhc_connections",
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

	badRequestSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := badRequestSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/rhc_connections",
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

	badRequestSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := badRequestSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

// TestPauseSourceAndItsApplications tests that the "pause source" endpoint sets all the applications and the source
// itself as paused, by modifying their "paused_at" column.
func TestPauseSourceAndItsApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/pause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourcePause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// TestResumeSourceAndItsApplications tests that the "unpause source" endpoint sets all the applications and the source
// itself as resumed, by setting their "paused_at" column as "NULL".
func TestResumeSourceAndItsApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceUnpause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// MockSender is just a mock which will allow us to control how the "RaiseEvent" function gets executed.
type MockSender struct {
}

// raiseEventFunc is an overrideable function which gets executed when the sender's "RaiseEvent" is called. This helps
// keeping the test logic inside each test.
var raiseEventFunc func(eventType string, payload []byte, headers []kafka.Header) error

// RaiseEvent is a placeholder function which simulates a call to the "RaiseEvent" function.
func (p MockSender) RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error {
	return raiseEventFunc(eventType, payload, headers)
}

// TestSourcePauseRaiseEventCheck tests that a proper "raise event" is raised when a source is paused.
func TestSourcePauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = events.EventStreamProducer{Sender: MockSender{}}

	var sourceRaiseEventCallCount int
	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		var err error

		switch eventType {
		case "Source.pause":
			err = sourceEventTestHelper(t, c, "Source.pause", payload, headers)

			sourceRaiseEventCallCount++
		case "Application.pause":
			err = applicationEventTestHelper(t, c, "Application.pause", eventType, payload, headers)

			applicationRaiseEventCallCount++
		default:
			t.Errorf(`incorrect event type when raising the event. Want "Source.pause" or "Application.pause", got "%s"`, eventType)
			err = errors.New(`incorrect event type raised`)
		}

		return err
	}

	err := SourcePause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are pausing a single source, therefore there should only have been 1 call to the "RaiseEvent" function.
		want := 1
		got := sourceRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the source event. Want "%d", got "%d"`, want, got)
		}
	}

	{
		// The source has 2 related application in the fixtures, so the "RaiseEvent" function should have been called
		// twice.
		want := 2
		got := applicationRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// TestSourceUnpauseRaiseEventCheck tests that a proper "raise event" is raised when a source is unpaused.
func TestSourceUnpauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = events.EventStreamProducer{Sender: MockSender{}}

	var sourceRaiseEventCallCount int
	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		var err error

		switch eventType {
		case "Source.unpause":
			err = sourceEventTestHelper(t, c, "Source.unpause", payload, headers)

			sourceRaiseEventCallCount++
		case "Application.unpause":
			err = applicationEventTestHelper(t, c, "Application.unpause", eventType, payload, headers)

			applicationRaiseEventCallCount++
		default:
			t.Errorf(`incorrect event type when raising the event. Want "Source.pause" or "Application.pause", got "%s"`, eventType)
			err = errors.New(`incorrect event type raised`)
		}

		return err
	}

	err := SourceUnpause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are resuming a single source, therefore there should only have been 1 call to the "RaiseEvent" function.
		want := 1
		got := sourceRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the source event. Want "%d", got "%d"`, want, got)
		}
	}

	{
		// The source has 2 related application in the fixtures, so the "RaiseEvent" function should have been called
		// twice.
		want := 2
		got := applicationRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// sourceEventTestHelper helps to test whether the received payload and the headers from the "RaiseEvent" function
// correspond to what we are expecting.
func sourceEventTestHelper(t *testing.T, c echo.Context, expectedEventType string, payload []byte, headers []kafka.Header) error {
	sourceDao, err := getSourceDao(c)
	if err != nil {
		t.Errorf(`could not get the source dao: %s`, err)
		return err
	}

	// Grab the source from the fixtures.
	expectedSource, err := sourceDao.GetById(&fixtures.TestSourceData[0].ID)
	if err != nil {
		t.Errorf(`could not fetch the source: %s`, err)
		return err
	}

	{
		// Turn the source into JSON.
		want, err := json.Marshal(expectedSource.ToEvent())
		if err != nil {
			t.Errorf(`error marshalling the event: %s`, err)
			return err
		}

		got := payload
		if !bytes.Equal(want, got) {
			t.Errorf(`incorrect payload received on raise event.Want "%s", got "%s"`, want, got)
			return err
		}
	}

	{
		// The header should contain the expected event type as well.
		want := expectedEventType
		got := string(headers[0].Value)
		if want != got {
			t.Errorf(`incorrect header on raise event. Want "%s", got "%s"`, want, got)
			return errors.New(`incorrect header on raise event`)
		}
	}

	return nil
}
