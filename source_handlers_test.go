package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceTypeSourceSubcollectionList(t *testing.T) {
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
	c.SetParamValues("1")

	err := SourceTypeListSource(c)
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

func TestApplicatioTypeListSourceSubcollectionList(t *testing.T) {
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
	c.SetParamValues("1")

	err := ApplicationTypeListSource(c)
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

		if s["name"] != "Source1" {
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

	if rec.Code != 200 {
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
	_, _ = dao.Delete(&id)
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

	if *src.AvailabilityStatusResponse.AvailabilityStatus != "available" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", *src.AvailabilityStatusResponse.AvailabilityStatus)
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
