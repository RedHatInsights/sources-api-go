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
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
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
	testutils.SkipIfNotRunningIntegrationTests(t)

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

func TestEndpointListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

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
	backupNotificationProducer := service.NotificationProducer
	service.NotificationProducer = &mocks.MockAvailabilityStatusNotificationProducer{}

	req := m.EndpointEditRequest{
		ReceptorNode:            util.StringRef("receptor_node"),
		Role:                    util.StringRef("role"),
		Scheme:                  util.StringRef("scheme"),
		Host:                    util.StringRef("host"),
		Path:                    util.StringRef("path"),
		CertificateAuthority:    util.StringRef("cert"),
		AvailabilityStatus:      util.StringRef("available"),
		AvailabilityStatusError: util.StringRef(""),
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
	c.Set("accountNumber", fixtures.TestTenantData[0].ExternalTenant)

	sourceEditHandlerWithNotifier := middleware.Notifier(EndpointEdit)
	err := sourceEditHandlerWithNotifier(c)
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
	notificationProducer, ok := service.NotificationProducer.(*mocks.MockAvailabilityStatusNotificationProducer)
	if !ok {
		t.Errorf("unable to cast notification producer")
	}

	emailNotificationInfo := &m.EmailNotificationInfo{ResourceDisplayName: "Endpoint",
		CurrentAvailabilityStatus:  "available",
		PreviousAvailabilityStatus: "unavailable",
		SourceName:                 "",
		SourceID:                   strconv.FormatInt(fixtures.TestSourceData[0].ID, 10),
		TenantID:                   strconv.FormatInt(fixtures.TestSourceData[0].TenantID, 10),
	}

	if !cmp.Equal(emailNotificationInfo, notificationProducer.EmailNotificationInfo) {
		t.Errorf("Invalid email notification data:")
		t.Errorf("Expected: %v Obtained: %v", emailNotificationInfo, notificationProducer.EmailNotificationInfo)
	}

	service.NotificationProducer = backupNotificationProducer

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
		AvailabilityStatus:      util.StringRef("available"),
		AvailabilityStatusError: util.StringRef(""),
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
		AvailabilityStatus:      util.StringRef("available"),
		AvailabilityStatusError: util.StringRef(""),
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

func TestEndpointDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/endpoints/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := EndpointDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusNoContent, rec.Code)
	}

	// Check that endpoint doesn't exist.
	c, rec = request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	notFoundEndpointGet := ErrorHandlingContext(EndpointGet)
	err = notFoundEndpointGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
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

func TestEndpointDeleteNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/endpoints/5789395389375",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("5789395389375")

	notFoundEndpointDelete := ErrorHandlingContext(EndpointDelete)
	err := notFoundEndpointDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}
