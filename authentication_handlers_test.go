package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestAuthenticationList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
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

	if len(out.Data) != 1 {
		t.Error("not enough objects passed back from DB")
	}

	auth1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if auth1["id"] != fixtures.TestAuthenticationData[0].ID {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestAuthenticationGet(t *testing.T) {
	uid := fixtures.TestAuthenticationData[0].ID

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

	if outAuthentication.ID != uid {
		t.Error("ghosts infected the return")
	}
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

	testutils.NotFoundTest(t, rec)
}

func TestAuthenticationCreate(t *testing.T) {
	requestBody := m.AuthenticationCreateRequest{
		Name:         "TestRequest",
		AuthType:     "test",
		Username:     "testUser",
		Password:     "123456",
		ResourceType: "Application",
		ResourceID:   1,
		SourceID:     1,
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

	if auth.ResourceType != "Application" {
		t.Errorf("Wrong resource type, wanted %v got %v", "Application", auth.ResourceType)
	}

	if auth.Username != "testUser" {
		t.Errorf("Wrong user name, wanted %v got %v", "testUser", auth.Username)
	}

	if auth.ResourceID != "1" {
		t.Errorf("Wrong resource ID, wanted %v got %v", 1, auth.ResourceID)
	}
}

func TestAuthenticationCreateBadRequest(t *testing.T) {
	requestBody := m.AuthenticationCreateRequest{
		Name:         "TestRequest",
		AuthType:     "test",
		Username:     "testUser",
		Password:     "123456",
		ResourceType: "InvalidType",
		ResourceID:   1,
		SourceID:     1,
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

	testutils.BadRequestTest(t, rec)
}

func TestAuthenticationUpdate(t *testing.T) {
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
	c.SetParamValues(fixtures.TestAuthenticationData[0].ID)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = AuthenticationUpdate(c)
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
}

func TestAuthenticationUpdateNotFound(t *testing.T) {
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

	notFoundAuthenticationUpdate := ErrorHandlingContext(AuthenticationUpdate)
	err = notFoundAuthenticationUpdate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestAuthenticationUpdateBadRequest(t *testing.T) {
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

	badRequestAuthenticationUpdate := ErrorHandlingContext(AuthenticationUpdate)
	err = badRequestAuthenticationUpdate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
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
	c.SetParamValues(fixtures.TestAuthenticationData[0].ID)
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

	testutils.NotFoundTest(t, rec)
}
