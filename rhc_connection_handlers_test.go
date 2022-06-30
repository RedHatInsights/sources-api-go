package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestRhcConnectionList(t *testing.T) {
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := RhcConnectionList(c)
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

	var wantCount int
	for _, rhc := range fixtures.TestRhcConnectionData {
		for _, srcRhc := range fixtures.TestSourceRhcConnectionData {
			if srcRhc.RhcConnectionId == rhc.ID && srcRhc.TenantId == tenantId {
				wantCount++
				break
			}
		}
	}

	if len(out.Data) != wantCount {
		t.Errorf("not enough objects passed back from DB, expected %d, got %d", wantCount, len(out.Data))
	}

	for _, rhcConnection := range out.Data {
		rhc, ok := rhcConnection.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}
		// Check that rhc connection belongs to correct tenant
		for _, srcRhc := range fixtures.TestSourceRhcConnectionData {
			if rhc["ID"] == fmt.Sprintf("%d", srcRhc.RhcConnectionId) {
				if srcRhc.TenantId != tenantId {
					t.Errorf("wrong tenant id in returned object, expected %d, got %d", tenantId, srcRhc.TenantId)
				}
				break
			}
		}
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestRhcConnectionListTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// For not existing tenant is expected that returned value
	// will be empty list and return code 200
	tenantId := fixtures.NotExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := RhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestRhcConnectionListTenantWithoutRhcConnections(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// For the tenant without rhc connections is expected that returned value
	// will be empty list and return code 200
	tenantId := int64(3)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := RhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestRhcConnectionListInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
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

	badRequestRhcConnectionList := ErrorHandlingContext(RhcConnectionList)
	err := badRequestRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionGetById(t *testing.T) {
	tenantId := int64(1)
	id := strconv.FormatInt(fixtures.TestRhcConnectionData[0].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+id,
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	err := RhcConnectionGetById(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("want %d, got %d", http.StatusOK, rec.Code)
	}

	var outRhcConnectionResponse model.RhcConnectionResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outRhcConnectionResponse)
	if err != nil {
		t.Error("Failed unmarshalling output")
	}

	if *outRhcConnectionResponse.RhcId != fixtures.TestRhcConnectionData[0].RhcId {
		t.Error("ghosts infected the return")
	}

	var outRhcId int64
	outRhcId, err = strconv.ParseInt(*outRhcConnectionResponse.Id, 10, 64)
	if err != nil {
		t.Error(err)
	}

	// check in fixtures that returned rhc connection belongs to the desired tenant
	for _, srcRhc := range fixtures.TestSourceRhcConnectionData {
		if srcRhc.RhcConnectionId == outRhcId {
			if srcRhc.TenantId != tenantId {
				t.Errorf("wrong tenant id, expected %d, got %d", tenantId, srcRhc.TenantId)
			}
			break
		}
	}
}

func TestRhcConnectionGetByIdMissingIdParam(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{},
	)

	badRequestRhcConnectionGetById := ErrorHandlingContext(RhcConnectionGetById)
	err := badRequestRhcConnectionGetById(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionGetByIdInvalidParam(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/xxx",
		nil,
		map[string]interface{}{},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestRhcConnectionGetById := ErrorHandlingContext(RhcConnectionGetById)
	err := badRequestRhcConnectionGetById(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestRhcConnectionGetByIdInvalidTenant tests that not found is returned for
// existing rhc connection but when tenant is not owner of rhc connection
func TestRhcConnectionGetByIdInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)
	rhcId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	notFoundRhcConnectionGetByUuid := ErrorHandlingContext(RhcConnectionGetById)
	err := notFoundRhcConnectionGetByUuid(c)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	templates.NotFoundTest(t, rec)
}

// TestRhcConnectionGetByIdTenantNotExists tests that not found is returned for
// not existing tenant
func TestRhcConnectionGetByIdTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	rhcId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	notFoundRhcConnectionGetByUuid := ErrorHandlingContext(RhcConnectionGetById)
	err := notFoundRhcConnectionGetByUuid(c)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionGetByIdNotFound(t *testing.T) {
	nonExistingId := "12345"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+nonExistingId,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(nonExistingId)

	notFoundRhcConnectionGetByUuid := ErrorHandlingContext(RhcConnectionGetById)
	err := notFoundRhcConnectionGetByUuid(c)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionCreate(t *testing.T) {
	requestBody := model.RhcConnectionCreateRequest{
		Extra:       nil,
		SourceIdRaw: fixtures.TestSourceData[1].ID,
		RhcId:       "12345",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = RhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

// TestRhcConnectionCreateTenantNotExists tests that not found is returned for
// not existing tenant
func TestRhcConnectionCreateTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId

	requestBody := model.RhcConnectionCreateRequest{
		Extra:       nil,
		SourceIdRaw: fixtures.TestSourceData[1].ID,
		RhcId:       "12345",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundRhcConnectionCreate := ErrorHandlingContext(RhcConnectionCreate)
	err = notFoundRhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestRhcConnectionCreateTenantNotOwnsSource tests that not found is returned for
// existing tenant who doesn't own the source
func TestRhcConnectionCreateTenantNotOwnsSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)

	requestBody := model.RhcConnectionCreateRequest{
		Extra:       nil,
		SourceIdRaw: fixtures.TestSourceData[1].ID,
		RhcId:       "12345",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundRhcConnectionCreate := ErrorHandlingContext(RhcConnectionCreate)
	err = notFoundRhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionCreateInvalidInput(t *testing.T) {
	requestBody := model.RhcConnectionCreateRequest{
		Extra:    nil,
		SourceId: fixtures.TestRhcConnectionData[0].ID,
		RhcId:    "", // this should make the validation fail.
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestRhcConnectionCreate := ErrorHandlingContext(RhcConnectionCreate)
	err = badRequestRhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionCreateNotExistingSource(t *testing.T) {
	requestBody := model.RhcConnectionCreateRequest{
		Extra:       nil,
		SourceIdRaw: "7238927389",
		RhcId:       "67890",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	sourceNotFoundRhcConnectionCreate := ErrorHandlingContext(RhcConnectionCreate)
	err = sourceNotFoundRhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionCreateRelationExists(t *testing.T) {
	requestBody := model.RhcConnectionCreateRequest{
		Extra:       nil,
		SourceIdRaw: fixtures.TestSourceData[0].ID,
		RhcId:       "a",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/rhc_connections",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestRhcConnectionCreate := ErrorHandlingContext(RhcConnectionCreate)
	err = badRequestRhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionEdit(t *testing.T) {
	tenantId := int64(1)
	rhcId := fixtures.TestRhcConnectionData[2].ID
	requestBody := model.RhcConnectionEditRequest{
		Extra: nil,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	err = RhcConnectionEdit(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// check in fixtures that changed rhc connection belongs to the desired tenant
	for _, srcRhc := range fixtures.TestSourceRhcConnectionData {
		if srcRhc.RhcConnectionId == rhcId {
			if srcRhc.TenantId != tenantId {
				t.Errorf("wrong tenant id, expected %d, got %d", tenantId, srcRhc.TenantId)
			}
			break
		}
	}
}

// TestRhcConnectionEditInvalidTenant tests situation when the tenant tries to
// edit existing not owned rhc connection
func TestRhcConnectionEditInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	rhcId := fixtures.TestRhcConnectionData[2].ID
	requestBody := model.RhcConnectionEditRequest{
		Extra: nil,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	notFoundRhcConnectionEdit := ErrorHandlingContext(RhcConnectionEdit)
	err = notFoundRhcConnectionEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionEditInvalidParam(t *testing.T) {
	invalidId := "xxx"

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/rhc_connections/"+invalidId,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.SetParamNames("id")
	c.SetParamValues(invalidId)

	badRequestRhcConnectionEdit := ErrorHandlingContext(RhcConnectionEdit)
	err := badRequestRhcConnectionEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionEditNotFound(t *testing.T) {
	invalidId := "12345"

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/rhc_connections/"+invalidId,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.SetParamNames("id")
	c.SetParamValues(invalidId)

	notFoundRhcConnectionEdit := ErrorHandlingContext(RhcConnectionEdit)
	err := notFoundRhcConnectionEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionDelete(t *testing.T) {
	id := strconv.FormatInt(fixtures.TestRhcConnectionData[2].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/rhc_connections/"+id,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	c.SetParamNames("id")
	c.SetParamValues(id)

	err := RhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestRhcConnectionDeleteInvalidParam(t *testing.T) {
	invalidId := "xxx"

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/rhc_connections/"+invalidId,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	c.SetParamNames("id")
	c.SetParamValues(invalidId)

	badRequestRhcConnectionDelete := ErrorHandlingContext(RhcConnectionDelete)
	err := badRequestRhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionDeleteMissingParam(t *testing.T) {
	id := strconv.FormatInt(fixtures.TestRhcConnectionData[2].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/rhc_connections/"+id,
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestRhcConnectionDelete := ErrorHandlingContext(RhcConnectionDelete)
	err := badRequestRhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestRhcConnectionDeleteInvalidTenant tests that not found err is returned
// when tenant tries to delete not owned rhc connection
func TestRhcConnectionDeleteInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	rhcId := fixtures.TestRhcConnectionData[2].ID

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	notFoundRhcConnectionDelete := ErrorHandlingContext(RhcConnectionDelete)
	err := notFoundRhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestRhcConnectionDeleteTenantNotExists tests that not found err is returned
// when tenant doesn't exist
func TestRhcConnectionDeleteTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(98304983)
	rhcId := fixtures.TestRhcConnectionData[2].ID

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d", rhcId),
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcId))

	notFoundRhcConnectionDelete := ErrorHandlingContext(RhcConnectionDelete)
	err := notFoundRhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionDeleteNotFound(t *testing.T) {
	nonExistingId := "12345"

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/rhc_connections/"+nonExistingId,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.SetParamNames("id")
	c.SetParamValues(nonExistingId)

	notFoundRhcConnectionDelete := ErrorHandlingContext(RhcConnectionDelete)
	err := notFoundRhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestRhcConnectionGetRelatedSources(t *testing.T) {
	tenantId := int64(1)
	rhcConnectionId := "2"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+rhcConnectionId+"/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(rhcConnectionId)

	err := RhcConnectionSourcesList(c)
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
		if 1 != len(out.Data) {
			t.Error("not enough objects passed back from DB")
		}
	} else {
		if len(fixtures.TestSourceData) != len(out.Data) {
			t.Error("not enough objects passed back from DB")
		}
	}

	for _, source := range out.Data {
		_, ok := source.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}
	}

	// Check that all sources belong to our tenant
	err = checkAllSourcesBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}
}

func TestRhcConnectionGetRelatedSourcesInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	rhcConnectionId := "2"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+rhcConnectionId+"/sources",
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

	c.SetParamNames("id")
	c.SetParamValues(rhcConnectionId)

	badRequestRhcConnectionSourcesList := ErrorHandlingContext(RhcConnectionSourcesList)
	err := badRequestRhcConnectionSourcesList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionGetRelatedSourcesInvalidParam(t *testing.T) {
	rhcConnectionId := "sss"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+rhcConnectionId+"/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(rhcConnectionId)

	badRequestRhcConnectionSourcesList := ErrorHandlingContext(RhcConnectionSourcesList)
	err := badRequestRhcConnectionSourcesList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestRhcConnectionGetRelatedSourcesNotFound(t *testing.T) {
	rhcConnectionId := "789678567"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+rhcConnectionId+"/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(rhcConnectionId)

	notFoundRhcConnectionSourcesList := ErrorHandlingContext(RhcConnectionSourcesList)
	err := notFoundRhcConnectionSourcesList(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestRhcConnectionGetRelatedSourcesInvalidTenant tests that not found err is returned
// when tenant tries to list sources for not owned rhc connection
func TestRhcConnectionGetRelatedSourcesInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)
	rhcConnectionId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d/sources", rhcConnectionId),
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcConnectionId))

	notFoundRhcConnectionSourcesList := ErrorHandlingContext(RhcConnectionSourcesList)
	err := notFoundRhcConnectionSourcesList(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestRhcConnectionGetRelatedSourcesTenantNotExists tests that not found err is returned
// when tenant doesn't exist
func TestRhcConnectionGetRelatedSourcesTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	rhcConnectionId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d/sources", rhcConnectionId),
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", rhcConnectionId))

	notFoundRhcConnectionSourcesList := ErrorHandlingContext(RhcConnectionSourcesList)
	err := notFoundRhcConnectionSourcesList(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}
