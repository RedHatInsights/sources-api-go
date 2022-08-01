package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func TestAuthenticationList(t *testing.T) {
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := AuthenticationList(c)
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

	var wantCount int
	for _, a := range fixtures.TestAuthenticationData {
		if a.TenantID == tenantId {
			wantCount++
		}
	}

	if out.Meta.Count != wantCount {
		t.Error("not enough objects passed back from DB")
	}

	if len(out.Data) != wantCount {
		t.Error("not enough objects passed back from DB")
	}

	auth1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if config.IsVaultOn() {
		if auth1["id"] != fixtures.TestAuthenticationData[0].ID {
			t.Errorf(`wrong authentication list fetched. Want authentications from the fixtures, got: %s`, auth1)
		}
	} else {
		outIdStr, ok := auth1["id"].(string)
		if !ok {
			t.Errorf(`Want "string", got "%s"`, auth1)
		}

		outId, err := strconv.ParseInt(outIdStr, 10, 64)
		if err != nil {
			t.Errorf(`The ID of the payload could not be converted to int64: %s`, err)
		}

		if fixtures.TestAuthenticationData[0].DbID != outId {
			t.Errorf(`wrong authentication list fetched. Want authentications from the fixtures, got: %s`, auth1)
		}
	}

	err = checkAllAuthenticationsBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestAuthenticationTenantNotExist tests that empty list is returned
// for not existing tenant
func TestAuthenticationTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := AuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestAuthenticationTenantWithoutAuthentications tests that empty list is returned
// for a tenant without authentications
func TestAuthenticationTenantWithoutAuthentications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err := AuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestAuthenticationListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": tenantId,
		},
	)

	badRequestAuthenticationList := ErrorHandlingContext(AuthenticationList)
	err := badRequestAuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestAuthenticationGet(t *testing.T) {
	tenantId := int64(1)
	originalSecretStore := conf.SecretStore

	// Run this test for secret store = [vault, database]
	for _, s := range []string{"vault", "database"} {
		conf.SecretStore = s

		var id string
		if config.IsVaultOn() {
			conf.SecretStore = "vault"
			id = fixtures.TestAuthenticationData[0].ID
		} else {
			id = strconv.FormatInt(fixtures.TestAuthenticationData[0].DbID, 10)
		}

		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/authentications/"+id,
			nil,
			map[string]interface{}{
				"tenantID": tenantId,
			},
		)

		c.SetParamNames("uid")
		c.SetParamValues(id)

		err := AuthenticationGet(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Error("Did not return 200")
		}

		var outAuthentication m.AuthenticationResponse
		err = json.Unmarshal(rec.Body.Bytes(), &outAuthentication)
		if err != nil {
			t.Error("Failed unmarshaling output")
		}

		if outAuthentication.ID != id {
			t.Errorf(`wrong authentication fetched. Want "%s", got "%s"`, id, outAuthentication.ID)
		}

		// Check the tenancy of returned authentication
		for _, a := range fixtures.TestAuthenticationData {
			var fixturesAuthId string
			if config.IsVaultOn() {
				fixturesAuthId = a.ID
			} else {
				fixturesAuthId = strconv.FormatInt(a.DbID, 10)
			}

			if fixturesAuthId == id {
				if a.TenantID != tenantId {
					t.Error("ghosts infected the return")
				}
				break
			}
		}
	}

	conf.SecretStore = originalSecretStore
}

func TestAuthenticationGetTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := fixtures.NotExistingTenantId
	uid := strconv.FormatInt(fixtures.TestAuthenticationData[0].DbID, 10)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications/"+uid,
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("uid")
	c.SetParamValues(uid)

	notFoundAuthenticationGet := ErrorHandlingContext(AuthenticationGet)
	err := notFoundAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestAuthenticationGetNotFound(t *testing.T) {
	uid := "abcdefg"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications/"+uid,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")
	c.SetParamValues(uid)

	notFoundAuthenticationGet := ErrorHandlingContext(AuthenticationGet)
	err := notFoundAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestAuthenticationCreate(t *testing.T) {
	tenantId := int64(1)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	// Test the creation of authentication for an application, fpr an endpoint
	// and for a source

	for _, resourceType := range []string{"Application", "Endpoint", "Source"} {
		requestBody := m.AuthenticationCreateRequest{
			Name:          util.StringRef("TestRequest"),
			AuthType:      "test",
			Username:      util.StringRef("testUser"),
			Password:      util.StringRef("123456"),
			ResourceType:  resourceType,
			ResourceIDRaw: 1,
		}

		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Error("Could not marshal JSON")
		}

		c, rec := request.CreateTestContext(
			http.MethodPost,
			"/api/sources/v3.1/authentications",
			bytes.NewReader(body),
			map[string]interface{}{
				"tenantID": tenantId,
			},
		)
		c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

		err = AuthenticationCreate(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusCreated {
			t.Errorf("Did not return 201. Body: %s", rec.Body.String())
		}

		auth := m.AuthenticationResponse{}
		raw, _ := io.ReadAll(rec.Body)
		err = json.Unmarshal(raw, &auth)
		if err != nil {
			t.Errorf("Failed to unmarshal application from response: %v", err)
		}

		if auth.ResourceType != resourceType {
			t.Errorf("Wrong resource type, wanted %v got %v", "Application", auth.ResourceType)
		}

		if auth.Username != "testUser" {
			t.Errorf("Wrong user name, wanted %v got %v", "testUser", auth.Username)
		}

		if auth.ResourceID != "1" {
			t.Errorf("Wrong resource ID, wanted %v got %v", 1, auth.ResourceID)
		}

		// Check the tenancy (only for integration tests)
		// and
		// Delete created authentication (it makes sense only for integration tests, in case of unit tests
		// the mock for auth create doesn't create new auth in the memory)
		if parser.RunningIntegrationTests {
			requestParams := dao.RequestParams{TenantID: &tenantId}
			authenticationDao := dao.GetAuthenticationDao(&requestParams)
			authOut, err := authenticationDao.GetById(auth.ID)
			if err != nil {
				t.Error(err)
			}
			if authOut.TenantID != tenantId {
				t.Errorf("authentication's tenant id = %d, expected %d", authOut.TenantID, tenantId)
			}

			_, err = authenticationDao.Delete(auth.ID)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func TestAuthenticationCreateBadRequestInvalidResourceType(t *testing.T) {
	requestBody := m.AuthenticationCreateRequest{
		Username:      util.StringRef("testUser"),
		Password:      util.StringRef("123456"),
		ResourceType:  "InvalidType",
		ResourceIDRaw: 1,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/authentications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestAuthenticationCreate := ErrorHandlingContext(AuthenticationCreate)
	err = badRequestAuthenticationCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestAuthenticationCreateResourceNotFound tests that bad request is returned when
// you try to create authentication for not existing resource
func TestAuthenticationCreateResourceNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	for _, resourceType := range []string{"Application", "Endpoint", "Source"} {

		requestBody := m.AuthenticationCreateRequest{
			Username:      util.StringRef("testUser"),
			Password:      util.StringRef("123456"),
			ResourceType:  resourceType,
			ResourceIDRaw: 54321,
		}

		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Error("Could not marshal JSON")
		}

		c, rec := request.CreateTestContext(
			http.MethodPost,
			"/api/sources/v3.1/authentications",
			bytes.NewReader(body),
			map[string]interface{}{
				"tenantID": tenantId,
			},
		)
		c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

		badRequestAuthenticationCreate := ErrorHandlingContext(AuthenticationCreate)
		err = badRequestAuthenticationCreate(c)
		if err != nil {
			t.Error(err)
		}

		templates.BadRequestTest(t, rec)
	}
}

func TestAuthenticationEdit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	backupNotificationProducer := service.NotificationProducer
	service.NotificationProducer = &mocks.MockAvailabilityStatusNotificationProducer{}

	newAvailabilityStatus := "new status"

	requestBody := m.AuthenticationEditRequest{
		AvailabilityStatus: &newAvailabilityStatus,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/authentications/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")
	if config.IsVaultOn() {
		c.SetParamValues(fixtures.TestAuthenticationData[0].ID)
	} else {
		id := strconv.FormatInt(fixtures.TestAuthenticationData[0].DbID, 10)
		c.SetParamValues(id)
	}
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

	authEditHandlerWithNotifier := middleware.Notifier(AuthenticationEdit)
	err = authEditHandlerWithNotifier(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Did not return 200. Body: %s", rec.Body.String())
	}

	auth := m.AuthenticationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &auth)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if auth.AvailabilityStatus != newAvailabilityStatus {
		t.Errorf("Wrong availability status, wanted %v got %v", newAvailabilityStatus, auth.AvailabilityStatus)
	}

	notificationProducer, ok := service.NotificationProducer.(*mocks.MockAvailabilityStatusNotificationProducer)
	if !ok {
		t.Errorf("unable to cast notification producer")
	}

	emailNotificationInfo := &m.EmailNotificationInfo{ResourceDisplayName: "Authentication",
		CurrentAvailabilityStatus:  newAvailabilityStatus,
		PreviousAvailabilityStatus: "available",
		SourceName:                 fixtures.TestSourceData[0].Name,
		SourceID:                   strconv.FormatInt(fixtures.TestSourceData[0].ID, 10),
		TenantID:                   strconv.FormatInt(fixtures.TestSourceData[0].TenantID, 10),
	}

	if !cmp.Equal(emailNotificationInfo, notificationProducer.EmailNotificationInfo) {
		t.Errorf("Invalid email notification data:")
		t.Errorf("Expected: %v Obtained: %v", emailNotificationInfo, notificationProducer.EmailNotificationInfo)
	}

	service.NotificationProducer = backupNotificationProducer
}

func TestAuthenticationEditNotFound(t *testing.T) {
	newAvailabilityStatus := "new status"

	requestBody := m.AuthenticationEditRequest{
		AvailabilityStatus: &newAvailabilityStatus,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/authentications/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")
	c.SetParamValues("not existing uid")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundAuthenticationEdit := ErrorHandlingContext(AuthenticationEdit)
	err = notFoundAuthenticationEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestAuthenticationEditBadRequest(t *testing.T) {
	requestBody :=
		`{
              "name": 10
         }`

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/authentications/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")
	c.SetParamValues(fixtures.TestAuthenticationData[0].ID)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestAuthenticationEdit := ErrorHandlingContext(AuthenticationEdit)
	err = badRequestAuthenticationEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestAuthenticationDelete(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")

	if config.IsVaultOn() {
		c.SetParamValues(fixtures.TestAuthenticationData[0].ID)
	} else {
		id := strconv.FormatInt(fixtures.TestAuthenticationData[0].DbID, 10)
		c.SetParamValues(id)
	}
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := AuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Did not return 204. Body: %s", rec.Body.String())
	}

	if rec.Body.Len() != 0 {
		t.Errorf("Response body is not nil")
	}
}

func TestAuthenticationDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("uid")
	c.SetParamValues("not existing uid")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundAuthenticationDelete := ErrorHandlingContext(AuthenticationDelete)
	err := notFoundAuthenticationDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// HELPERS:

// checkAllAuthenticationsBelongToTenant checks that all returned authentications belong to given tenant
func checkAllAuthenticationsBelongToTenant(tenantId int64, authentications []interface{}) error {
	for _, authOut := range authentications {
		authOutId := authOut.(map[string]interface{})["id"].(string)
		// find authentication in fixtures and check the tenant id
		for _, auth := range fixtures.TestAuthenticationData {
			if authOutId == auth.ID {
				if auth.TenantID != tenantId {
					return fmt.Errorf("expected tenant id = %d, got %d", tenantId, auth.TenantID)
				}
			}
		}
	}
	return nil
}
