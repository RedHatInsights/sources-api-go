package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceEndpointSubcollectionList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/endpoints",
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

	err := SourceListEndpoint(c)
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

	e1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if e1["id"] != "1" {
		t.Error("ghosts infected the return")
	}

	e2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if e2["id"] != "2" {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceEndpointSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/983749387/endpoints",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("983749387")

	notFoundSourceListEndpoint := ErrorHandlingContext(SourceListEndpoint)
	err := notFoundSourceListEndpoint(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceEndpointSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/endpoints",
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

	badRequestSourceListEndpoint := ErrorHandlingContext(SourceListEndpoint)
	err := badRequestSourceListEndpoint(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceEndpointSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/endpoints",
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

	badRequestSourceListEndpoint := ErrorHandlingContext(SourceListEndpoint)
	err := badRequestSourceListEndpoint(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceEndpointSubcollectionListWithOffsetAndLimit(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	sourceID := int64(1)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	// How many endpoints with given source ID is in fixtures
	var wantEndpointsCount int
	for _, e := range fixtures.TestEndpointData {
		if e.SourceID == sourceID {
			wantEndpointsCount++
		}
	}

	for _, i := range testData {

		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/sources/1/endpoints",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		c.SetParamNames("source_id")
		c.SetParamValues(fmt.Sprintf("%d", sourceID))

		err := SourceListEndpoint(c)
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

		if out.Meta.Count != wantEndpointsCount {
			t.Errorf("count not set correctly, got %d, want %d", out.Meta.Count, wantEndpointsCount)
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := wantEndpointsCount - i["offset"]
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

func TestEndpointList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	err := EndpointList(c)
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

	if len(out.Data) != len(fixtures.TestEndpointData) {
		t.Error("not enough objects passed back from DB")
	}

	SortByStringValueOnKey("id", out.Data)

	e1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if e1["id"] != "1" {
		t.Error("ghosts infected the return")
	}

	e2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if e2["id"] != "2" {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestEndpointListBadRequestInvalidFilter(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints",
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

	badRequestEndpointList := ErrorHandlingContext(EndpointList)
	err := badRequestEndpointList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestEndpointListWithOffsetAndLimit(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

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
			"/api/sources/v3.1/endpoints",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		err := EndpointList(c)
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

		if out.Meta.Count != len(fixtures.TestEndpointData) {
			t.Errorf("count not set correctly")
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := len(fixtures.TestEndpointData) - i["offset"]
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

func TestEndpointGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := EndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outEndpoint m.EndpointResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outEndpoint)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}
}

func TestEndpointGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/970283452983",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("970283452983")

	notFoundEndpointGet := ErrorHandlingContext(EndpointGet)
	err := notFoundEndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestEndpointGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestEndpointGet := ErrorHandlingContext(EndpointGet)
	err := badRequestEndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// Tests that the endpoint is properly creating "endpoints" and returning a 201 code.
func TestEndpointCreate(t *testing.T) {
	receptorNode := "receptorNode"
	scheme := "scheme"
	port := 443
	verifySsl := true
	certificateAuthority := "Let's Encrypt"

	requestBody := m.EndpointCreateRequest{
		Default:              false,
		ReceptorNode:         &receptorNode,
		Role:                 "role",
		Scheme:               &scheme,
		Host:                 "example.com",
		Port:                 &port,
		Path:                 "",
		VerifySsl:            &verifySsl,
		CertificateAuthority: &certificateAuthority,
		AvailabilityStatus:   m.Available,
		SourceIDRaw:          fixtures.TestSourceData[0].ID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/endpoints",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = EndpointCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 201 {
		t.Errorf("want 201, got %d", rec.Code)
	}
}

// TestEndpointCreateBadRequest tests that if a bad input is given, the endpoint returns a 400 response.
func TestEndpointCreateBadRequest(t *testing.T) {
	requestBody := m.EndpointCreateRequest{
		Host: "hello world",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/endpoints",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestEndpointCreate := ErrorHandlingContext(EndpointCreate)
	err = badRequestEndpointCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestEndpointEdit(t *testing.T) {
	req := m.EndpointEditRequest{
		ReceptorNode:            request.PointerToString("receptor_node"),
		Role:                    request.PointerToString("role"),
		Scheme:                  request.PointerToString("scheme"),
		Host:                    request.PointerToString("host"),
		Path:                    request.PointerToString("path"),
		CertificateAuthority:    request.PointerToString("cert"),
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/endpoints/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := EndpointEdit(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Errorf("Wrong return code, expected %v got %v", 200, rec.Code)
	}

	app := m.EndpointResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &app)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if *app.AvailabilityStatus != "available" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.ReceptorNode != "receptor_node" {
		t.Errorf("Wrong receptor node, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.Role != "role" {
		t.Errorf("Wrong role, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.Scheme != "scheme" {
		t.Errorf("Wrong scheme, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.Host != "host" {
		t.Errorf("Wrong host, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.Path != "path" {
		t.Errorf("Wrong path, wanted %v got %v", "available", *app.AvailabilityStatus)
	}

	if *app.CertificateAuthority != "cert" {
		t.Errorf("Wrong certificate authority, wanted %v got %v", "available", *app.AvailabilityStatus)
	}
}

func TestEndpointEditNotFound(t *testing.T) {
	req := m.EndpointEditRequest{
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/endpoints/9764567834",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9764567834")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundApplicationEdit := ErrorHandlingContext(EndpointEdit)
	err := notFoundApplicationEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestEndpointEditBadRequest(t *testing.T) {
	req := m.EndpointEditRequest{
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/endpoints/xxx",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestEndpointEdit := ErrorHandlingContext(EndpointEdit)
	err := badRequestEndpointEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestEndpointDeleteBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/endpoints/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestEndpointDelete := ErrorHandlingContext(EndpointDelete)
	err := badRequestEndpointDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}
