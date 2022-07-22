package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/kafka"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestApplicationAuthenticationList(t *testing.T) {
	tenantId := int64(1)
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := ApplicationAuthenticationList(c)
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

	var wantAppAuthCount int
	for _, a := range fixtures.TestApplicationAuthenticationData {
		if a.TenantID == tenantId {
			wantAppAuthCount++
		}
	}

	if len(out.Data) != wantAppAuthCount {
		t.Error("not enough objects passed back from DB")
	}

	appAuth, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if appAuth["id"] != "2" {
		t.Error("ghosts infected the return")
	}

	if appAuth["application_id"] != "5" {
		t.Error("ghosts infected the return")
	}

	// This is working only when SECRET_STORE=database
	// In app auth table we don't have column for vault path and we are checking this
	// when we creating the response, so then we get different results for unit tests
	// (auth ID = hash from vault path) and for integration tests (auth ID is db ID because
	// vault path column is missing in db)
	if conf.SecretStore == "database" {
		authID := strconv.Itoa(int(fixtures.TestAuthenticationData[3].DbID))
		if appAuth["authentication_id"].(string) != authID {
			t.Error("ghosts infected the return")
		}
	}

	err = checkAllApplicationAuthenticationsBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestApplicationAuthenticationListTenantNotExist tests that empty list is returned
// for not existing tenant
func TestApplicationAuthenticationListTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := ApplicationAuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestApplicationAuthenticationListTenantWithoutAppAuths tests that empty list is returned
// for tenant without application authentication
func TestApplicationAuthenticationListTenantWithoutAppAuths(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := ApplicationAuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestApplicationAuthenticationListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
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

	badRequestApplicationAuthenticationList := ErrorHandlingContext(ApplicationAuthenticationList)
	err := badRequestApplicationAuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationGet(t *testing.T) {
	tenantId := int64(1)
	appAuthId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	err := ApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out m.ApplicationAuthenticationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	// This is working only when SECRET_STORE=database
	// In app auth table we don't have column for vault path and we are checking this
	// when we creating the response, so then we get different results for unit tests
	// (auth ID = hash from vault path) and for integration tests (auth ID is db ID because
	// vault path column is missing in db)
	if conf.SecretStore == "database" {
		authID := strconv.Itoa(int(fixtures.TestAuthenticationData[3].DbID))
		if out.AuthenticationID != authID {
			t.Error("ghosts infected the return")
		}
	}

	if out.ApplicationID != "5" {
		t.Error("ghosts infected the return")
	}

	// Check the tenancy of returned app auth
	for _, appAuth := range fixtures.TestApplicationAuthenticationData {
		if fmt.Sprintf("%d", appAuth.ID) == out.ID {
			if appAuth.TenantID != tenantId {
				t.Errorf("returned app auth not belong to the tenant, expected tenantd id %d, got %d", tenantId, appAuth.TenantID)
			}
		}
	}
}

// TestApplicationAuthenticationGetInvalidTenant tests that not found is returned
// for existing app auth id but with invalid tenant
func TestApplicationAuthenticationGetInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	appAuthId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	notFoundApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := notFoundApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestApplicationAuthenticationGetTenantNotExist tests that not found is returned
// for not existing tenant
func TestApplicationAuthenticationGetTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	appAuthId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	notFoundApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := notFoundApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/13094830948",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("13094830948")

	notFoundApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := notFoundApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	// Create own test data - source with application and authentication
	// Create a source
	daoParams := dao.RequestParams{TenantID: &tenantId}
	sourceDao := dao.GetSourceDao(&daoParams)

	uid := "bd2ba6d6-4630-40e2-b829-cf09b03bdb9f"
	src := m.Source{
		Name:         "Source for TestApplicationAuthenticationCreate()",
		SourceTypeID: 1,
		Uid:          &uid,
	}

	err := sourceDao.Create(&src)
	if err != nil {
		t.Errorf("source not created correctly: %s", err)
	}

	// Create an application
	applicationDao := dao.GetApplicationDao(&daoParams)

	app := m.Application{
		SourceID:          src.ID,
		ApplicationTypeID: 1,
		Extra:             []byte(`{"Name": "app for TestApplicationAuthenticationCreate()"}`),
	}

	err = applicationDao.Create(&app)
	if err != nil {
		t.Errorf("application not created correctly: %s", err)
	}

	// Create an authentication for the application
	authenticationDao := dao.GetAuthenticationDao(&daoParams)

	authNameForApp := "authentication for TestApplicationAuthenticationCreate()"
	auth := m.Authentication{
		Name:         &authNameForApp,
		ResourceType: "Application",
		ResourceID:   app.ID,
		TenantID:     tenantId,
		SourceID:     src.ID,
	}

	err = authenticationDao.Create(&auth)
	if err != nil {
		t.Errorf("authentication for application not created correctly: %s", err)
	}

	// Create a test context and call the ApplicationAuthenticationCreate()
	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    app.ID,
		AuthenticationIDRaw: auth.DbID,
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	err = ApplicationAuthenticationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 201 {
		t.Errorf("Wrong response code, got %v wanted %v", rec.Code, 201)
	}

	var out m.ApplicationAuthenticationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	appAuthID, err := strconv.ParseInt(out.ID, 10, 64)
	if err != nil {
		t.Error(err)
	}
	appAuthDao := dao.GetApplicationAuthenticationDao(&daoParams)
	appAuthOut, err := appAuthDao.GetById(&appAuthID)
	if err != nil {
		t.Error(err)
	}

	// Check the tenancy of all related objects
	if appAuthOut.TenantID != tenantId {
		t.Errorf("application authentication with id %d should belong to tenant with id %d, but got %d", appAuthOut.ID, tenantId, appAuthOut.TenantID)
	}
	if app.TenantID != tenantId {
		t.Errorf("application with id %d should belong to tenant with id %d, but got %d", app.ID, tenantId, app.TenantID)
	}
	if auth.TenantID != tenantId {
		t.Errorf("authentication with id %d should belong to tenant with id %d, but got %d", auth.DbID, tenantId, auth.TenantID)
	}

	// Delete the created test data
	err = service.DeleteCascade(&tenantId, nil, "Source", src.ID, []kafka.Header{})
	if err != nil {
		t.Error(err)
	}
}

func TestApplicationAuthenticationCreateBadAppId(t *testing.T) {
	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    "abcd",
		AuthenticationIDRaw: 7,
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationCreate)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationCreateBadAuthId(t *testing.T) {
	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    7,
		AuthenticationIDRaw: "abcd",
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationCreate)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	// Create own test data - to be independent on fixtures
	// Create a source
	daoParams := dao.RequestParams{TenantID: &tenantId}
	sourceDao := dao.GetSourceDao(&daoParams)

	uid := "bd2ba6d6-4630-40e2-b829-cf09b03bdb9f"
	src := m.Source{
		Name:         "Source for TestApplicationAuthenticationDelete()",
		SourceTypeID: 1,
		Uid:          &uid,
	}

	err := sourceDao.Create(&src)
	if err != nil {
		t.Errorf("source not created correctly: %s", err)
	}

	// Create an application
	applicationDao := dao.GetApplicationDao(&daoParams)

	app := m.Application{
		SourceID:          src.ID,
		ApplicationTypeID: 1,
		Extra:             []byte(`{"Name": "app for TestApplicationAuthenticationDelete()"}`),
	}

	err = applicationDao.Create(&app)
	if err != nil {
		t.Errorf("application not created correctly: %s", err)
	}

	// Create an authentication for the application
	authDaoParams := dao.RequestParams{TenantID: &tenantId}
	authenticationDao := dao.GetAuthenticationDao(&authDaoParams)

	authNameForApp := "authentication for TestApplicationAuthenticationDelete()"
	auth := m.Authentication{
		Name:         &authNameForApp,
		ResourceType: "Application",
		ResourceID:   app.ID,
		TenantID:     tenantId,
		SourceID:     src.ID,
	}

	err = authenticationDao.Create(&auth)
	if err != nil {
		t.Errorf("authentication for application not created correctly: %s", err)
	}

	// Create an application authentication
	appAuthDao := dao.GetApplicationAuthenticationDao(&daoParams)
	appAuth := m.ApplicationAuthentication{
		ApplicationID:    app.ID,
		AuthenticationID: auth.DbID,
	}
	err = appAuthDao.Create(&appAuth)
	if err != nil {
		t.Errorf("application authentication not created correctly: %s", err)
	}

	// Create text context and call the ApplicationAuthenticationDelete() handler
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/application_authentications/300",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", appAuth.ID))
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = ApplicationAuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Did not return 204. Body: %s", rec.Body.String())
	}

	if rec.Body.Len() != 0 {
		t.Errorf("Response body is not nil")
	}

	// Check that application authentication doesn't exist
	_, err = appAuthDao.GetById(&appAuth.ID)
	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf("expected Not found error, got %s", err)
	}

	// Delete the created test data
	err = service.DeleteCascade(&tenantId, nil, "Source", src.ID, []kafka.Header{})
	if err != nil {
		t.Error(err)
	}
}

// TestApplicationAuthenticationDeleteInvalidTenant tests that not found is returned
// for tenant who doesn't own the app auth
func TestApplicationAuthenticationDeleteInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	appAuthId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/application_authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundApplicationAuthenticationDelete := ErrorHandlingContext(ApplicationAuthenticationDelete)
	err := notFoundApplicationAuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/application_authentications/1234523452542",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.SetParamNames("id")
	c.SetParamValues("1234523452542")

	notFoundApplicationAuthenticationDelete := ErrorHandlingContext(ApplicationAuthenticationDelete)
	err := notFoundApplicationAuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationDeleteBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/application_authentications/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationAuthenticationDelete := ErrorHandlingContext(ApplicationAuthenticationDelete)
	err := badRequestApplicationAuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationListAuthentications(t *testing.T) {
	testutils.SkipIfNotSecretStoreDatabase(t)
	tenantId := int64(1)
	appAuthId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_authentication_id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	err := ApplicationAuthenticationListAuthentications(c)
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

	if out.Meta.Count != 1 {
		t.Error("count not set correctly")
	}

	auth, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as an authentication")
	}

	authWant := fixtures.TestAuthenticationData[3]
	if auth["id"] != fmt.Sprintf("%d", authWant.DbID) {
		t.Errorf("wrong authentication id, want %d, got %s", authWant.DbID, auth["id"])
	}

	if auth["resource_type"] != authWant.ResourceType {
		t.Errorf("wrong authentication resource type, want %s, got %s", authWant.ResourceType, auth["resource_type"])
	}

	if auth["resource_id"] != fmt.Sprintf("%d", authWant.ResourceID) {
		t.Errorf("wrong authentication resource type, want %d, got %s", authWant.ResourceID, auth["resource_id"])
	}

	// Check the tenancy of returned authentication
	for _, a := range fixtures.TestAuthenticationData {
		if fmt.Sprintf("%d", a.DbID) == auth["id"] {
			if a.TenantID != tenantId {
				t.Errorf("expected tenant id = %d, got tenant id = %d for authentication DbId = %d", tenantId, a.TenantID, a.DbID)
			}
		}
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestApplicationAuthenticationListAuthenticationsTenantNotExist tests that not found
// is returned for not existing tenant
func TestApplicationAuthenticationListAuthenticationsTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	appAuthId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/2/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_authentication_id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	notFoundAppAuthListAuths := ErrorHandlingContext(ApplicationAuthenticationListAuthentications)
	err := notFoundAppAuthListAuths(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestApplicationAuthenticationListAuthenticationsInvalidTenant tests that not found
// is returned for valid tenant who doesn't own the app auth
func TestApplicationAuthenticationListAuthenticationsInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(2)
	appAuthId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/2/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_authentication_id")
	c.SetParamValues(fmt.Sprintf("%d", appAuthId))

	notFoundAppAuthListAuths := ErrorHandlingContext(ApplicationAuthenticationListAuthentications)
	err := notFoundAppAuthListAuths(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationListAuthenticationsNotFound(t *testing.T) {
	testutils.SkipIfNotSecretStoreDatabase(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/78967899/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_authentication_id")
	c.SetParamValues("78967899")

	notFoundAppAuthListAuths := ErrorHandlingContext(ApplicationAuthenticationListAuthentications)
	err := notFoundAppAuthListAuths(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationListAuthenticationsBadRequest(t *testing.T) {
	testutils.SkipIfNotSecretStoreDatabase(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/xxx/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_authentication_id")
	c.SetParamValues("xxx")

	badRequestAppAuthListAuths := ErrorHandlingContext(ApplicationAuthenticationListAuthentications)
	err := badRequestAppAuthListAuths(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// HELPERS:

// checkAllApplicationAuthenticationsBelongToTenant checks that all returned application authentications
// belongs to given tenant
func checkAllApplicationAuthenticationsBelongToTenant(tenantId int64, appAuths []interface{}) error {
	// For every returned application authentication
	for _, appAuthOut := range appAuths {
		appAuthOutId, err := strconv.ParseInt(appAuthOut.(map[string]interface{})["id"].(string), 10, 64)
		if err != nil {
			return err
		}
		// find application authentication in fixtures and check the tenant id
		for _, appAuth := range fixtures.TestApplicationAuthenticationData {
			if appAuthOutId == appAuth.ID {
				if appAuth.TenantID != tenantId {
					return fmt.Errorf("expected tenant id = %d, got %d", tenantId, appAuth.TenantID)
				}
			}
		}
	}
	return nil
}
