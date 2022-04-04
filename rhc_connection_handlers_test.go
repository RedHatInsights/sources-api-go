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
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestRhcConnectionList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
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

	if len(out.Data) != len(fixtures.TestRhcConnectionData) {
		t.Error("not enough objects passed back from DB")
	}

	for _, rhcConnection := range out.Data {
		_, ok := rhcConnection.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}

	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestRhcConnectionListWithOffsetAndLimit(t *testing.T) {
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
			"/api/sources/v3.1/rhc_connections",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
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

		if out.Meta.Limit != i["limit"] {
			t.Error("limit not set correctly")
		}

		if out.Meta.Offset != i["offset"] {
			t.Error("offset not set correctly")
		}

		if out.Meta.Count != len(fixtures.TestRhcConnectionData) {
			t.Errorf("count not set correctly, want %d, got %d", len(fixtures.TestRhcConnectionData), out.Meta.Count)
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := len(fixtures.TestRhcConnectionData) - i["offset"]
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

func TestRhcConnectionGetById(t *testing.T) {
	id := strconv.FormatInt(fixtures.TestRhcConnectionData[0].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections/"+id,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
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
}

func TestRhcConnectionGetByIdMissingIdParam(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/rhc_connections",
		nil,
		map[string]interface{}{},
	)

	err := RhcConnectionGetById(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rec.Code)
	}
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("want %d, got %d", http.StatusNotFound, rec.Code)
	}
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

	err = RhcConnectionCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestRhcConnectionUpdate(t *testing.T) {
	requestBody := model.RhcConnectionUpdateRequest{
		Extra: nil,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	id := strconv.FormatInt(fixtures.TestRhcConnectionData[2].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/rhc_connections/"+id,
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	c.SetParamNames("id")
	c.SetParamValues(id)

	err = RhcConnectionUpdate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRhcConnectionUpdateNotFound(t *testing.T) {
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

	notFoundRhcConnectionUpdate := ErrorHandlingContext(RhcConnectionUpdate)
	err := notFoundRhcConnectionUpdate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
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

func TestRhcConnectionDeleteMissingParam(t *testing.T) {
	id := strconv.FormatInt(fixtures.TestRhcConnectionData[2].ID, 10)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/rhc_connections/"+id,
		nil,
		map[string]interface{}{},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := RhcConnectionDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Want status code %d. Got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestRhcConnectionGetRelatedSources(t *testing.T) {
	rhcConnectionId := "2"

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
}

func TestRhcConnectionSourcesListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 2},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	rhcConnectionId := fixtures.TestRhcConnectionData[0].ID

	// Getting count of sources with given rhc connection id
	var wantSourcesCount int
	for _, i := range fixtures.TestSourceRhcConnectionData {
		if i.RhcConnectionId == rhcConnectionId {
			wantSourcesCount++
		}
	}

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			fmt.Sprintf("/api/sources/v3.1/rhc_connections/%d/sources", rhcConnectionId),
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprintf("%d", rhcConnectionId))

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
