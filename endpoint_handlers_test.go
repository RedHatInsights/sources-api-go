package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
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
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
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

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
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
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

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
	testutils.SkipIfNotSecretStoreDatabase(t)

	// EndpointDelete() uses cascade delete - deleted is not only
	// endpoint itself but related authentications too.
	// This test creates own test data to not affect other tests.

	// Create a source
	tenantID := fixtures.TestTenantData[0].Id
	sourceDao := dao.GetSourceDao(&tenantID)

	src := m.Source{
		Name:         "Source for TestApplicationDelete()",
		SourceTypeID: 1,
	}

	err := sourceDao.Create(&src)
	if err != nil {
		t.Errorf("source not created correctly: %s", err)
	}

	// Create an endpoint
	endpointDao := dao.GetEndpointDao(&tenantID)

	role := "new role"
	endpoint := m.Endpoint{
		SourceID: src.ID,
		TenantID: tenantID,
		Role:     &role,
	}

	err = endpointDao.Create(&endpoint)
	if err != nil {
		t.Errorf("endpoint not created correctly: %s", err)
	}

	// Create an authentication for endpoint
	authenticationDao := dao.GetAuthenticationDao(&tenantID)

	authName3 := "authentication for endpoint"
	auth := m.Authentication{
		Name:         &authName3,
		ResourceType: "Endpoint",
		ResourceID:   endpoint.ID,
		TenantID:     tenantID,
		SourceID:     src.ID,
	}

	err = authenticationDao.Create(&auth)
	if err != nil {
		t.Errorf("authentication for endpoint not created correctly: %s", err)
	}

	// Create test context and call the ApplicationDelete()
	id := fmt.Sprintf("%d", endpoint.ID)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		fmt.Sprintf("/api/sources/v3.1/endpoints/%s", id),
		nil,
		map[string]interface{}{
			"tenantID": tenantID,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = EndpointDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusNoContent, rec.Code)
	}

	// Check that endpoint doesn't exist
	_, err = endpointDao.GetById(&endpoint.ID)
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf("expected 'endpoint not found', got %s", err)
	}

	// Check that authentication doesn't exist
	_, err = authenticationDao.GetById(auth.ID)
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf("expected 'authentication not found', got %s", err)
	}

	// Clean up - delete created source
	_, err = sourceDao.Delete(&src.ID)
	if err != nil {
		t.Errorf("source not deleted correctly: %s", err)
	}
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

// TestEndpointEditPaused tests that an endpoint can be edited even if it is paused, if the payload is right. Runs on
// unit tests by swapping the mock endpoint's DAO to one that simulates that the endpoints are paused.
func TestEndpointEditPaused(t *testing.T) {
	validDate := time.Now().Format(util.RecordDateTimeFormat)

	req := m.ResourceEditPausedRequest{
		AvailabilityStatus:      util.StringRef("available"),
		AvailabilityStatusError: util.StringRef(""),
		LastAvailableAt:         &validDate,
		LastCheckedAt:           &validDate,
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

	// Make sure we are using the "NoUnknownFieldsBinder".
	backupBinder := c.Echo().Binder
	c.Echo().Binder = &NoUnknownFieldsBinder{}

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("accountNumber", fixtures.TestTenantData[0].ExternalTenant)

	// Store the original "getEndopintDao" function to restore it later.
	backupGetEndpointDao := getEndpointDao
	getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) {
		return &dao.MockEndpointDao{Endpoints: fixtures.TestEndpointData}, nil
	}

	// Set the fixture endpoint as "paused".
	pausedAt := time.Now()
	fixtures.TestEndpointData[0].PausedAt = &pausedAt

	badRequestEndpointEdit := ErrorHandlingContext(EndpointEdit)
	err := badRequestEndpointEdit(c)

	// Revert the fixture endpoint to its default value.
	fixtures.TestEndpointData[0].PausedAt = nil
	if err != nil {
		t.Errorf(`unexpected error when editing a paused endpoint: %s`, err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusOK, rec.Code)
	}

	// Restore the original "getEndpointDao" function.
	getEndpointDao = backupGetEndpointDao

	// Restore the binder to not affect any other tests.
	c.Echo().Binder = backupBinder
}

// TestEndpointEditPausedInvalidFields tests that a "bad request" response is returned when a paused endpoint is tried
// to be updated when the payload has not allowed fields. Runs on unit tests by swapping the mock endpoint's DAO to one
// that simulates that the endpoints are paused.
func TestEndpointEditPausedInvalidFields(t *testing.T) {
	availabilityStatus := "available"
	req := m.EndpointEditRequest{
		AvailabilityStatus: &availabilityStatus,
		Scheme:             util.StringRef("scheme"),
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

	// Make sure we don't accept the "Scheme" field we set up above.
	backupBinder := c.Echo().Binder
	c.Echo().Binder = &NoUnknownFieldsBinder{}

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("accountNumber", fixtures.TestTenantData[0].ExternalTenant)

	// Set the fixture endpoint as "paused".
	pausedAt := time.Now()
	fixtures.TestEndpointData[0].PausedAt = &pausedAt

	// Store the original "getEndopintDao" function to restore it later.
	backupGetEndpointDao := getEndpointDao
	getEndpointDao = func(c echo.Context) (dao.EndpointDao, error) {
		return &dao.MockEndpointDao{Endpoints: fixtures.TestEndpointData}, nil
	}

	badRequestEndpointEdit := ErrorHandlingContext(EndpointEdit)
	err := badRequestEndpointEdit(c)

	// Revert the fixture endpoint to its default value.
	fixtures.TestEndpointData[0].PausedAt = nil
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusBadRequest, rec.Code)
	}

	got, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Errorf(`error reading the response: %s`, err)
	}

	want := []byte("scheme")
	if !bytes.Contains(got, want) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, want, err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusBadRequest, rec.Code)
	}

	// Restore the original "getEndpointDao" function.
	getEndpointDao = backupGetEndpointDao

	// Restore the binder to not affect any other tests.
	c.Echo().Binder = backupBinder
}

func TestEndpointListAuthentications(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("endpoint_id")
	c.SetParamValues("1")

	err := EndpointListAuthentications(c)
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

	auth1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if auth1["resource_type"] != "Endpoint" {
		t.Error("ghosts infected the return")
	}

	if auth1["resource_id"] != "1" {
		t.Error("ghosts infected the return")
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestEndpointListAuthenticationsBadRequestInvalidEndpointId(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/xxx/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("endpoint_id")
	c.SetParamValues("xxx")

	badRequestEndpointListAuthentications := ErrorHandlingContext(EndpointListAuthentications)
	err := badRequestEndpointListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestEndpointListAuthenticationsBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/xxx/authentications",
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

	c.SetParamNames("endpoint_id")
	c.SetParamValues("xxx")

	badRequestEndpointListAuthentications := ErrorHandlingContext(EndpointListAuthentications)
	err := badRequestEndpointListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestEndpointListAuthenticationsNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/endpoints/09834098349/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("endpoint_id")
	c.SetParamValues("09834098349")

	notFoundEndpointListAuthentications := ErrorHandlingContext(EndpointListAuthentications)
	err := notFoundEndpointListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}
