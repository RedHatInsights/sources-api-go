package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func TestSecretCreateNameExistInCurrentTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantId := int64(1)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	name := "test-name"
	authType := "auth-type"
	userName := "test-name"
	password := "123456"
	secretExtra := map[string]interface{}{"extra": map[string]interface{}{"extra": "params"}}

	secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra)
	_, rec, err := createSecretRequest(t, secretCreateRequest, &tenantId, false)
	if err != nil {
		t.Error(err)
	}

	secret := parseSecretResponse(t, rec)

	_, _, err = createSecretRequest(t, secretCreateRequest, &tenantId, false)

	if err != nil && err.Error() != "bad request: secret name "+name+" exists in current tenant" {
		t.Error(err)
	}

	cleanSecretByID(t, secret.ID, &tenantId)
}

func TestSecretCreateEmptyName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantId := int64(1)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	name := ""
	authType := "auth-type"
	userName := "test-name"
	password := "123456"
	secretExtra := map[string]interface{}{"extra": map[string]interface{}{"extra": "params"}}

	secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra)

	_, _, err := createSecretRequest(t, secretCreateRequest, &tenantId, false)

	if err.Error() != "bad request: secret name have to be populated" {
		t.Error(err)
	}
}

func TestSecretCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantId := int64(1)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	name := "TestName"
	authType := "auth-type"
	userName := "test-name"
	password := "123456"
	secretExtra := map[string]interface{}{"extra": map[string]interface{}{"extra": "params"}}

	secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra)

	_, rec, err := createSecretRequest(t, secretCreateRequest, &tenantId, false)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Did not return 201. Body: %s", rec.Body.String())
	}

	secret := parseSecretResponse(t, rec)

	stringMatcher(t, "secret name", secret.Name, name)
	stringMatcher(t, "secret user name", secret.Username, userName)
	stringMatcher(t, "secret auth type", secret.AuthType, authType)

	secretOut := fetchSecretFromDB(t, secret.ID, tenantId)

	int64Matcher(t, "secret tenant id", secretOut.TenantID, tenantId)
	stringMatcher(t, "secret name", *secretOut.Name, name)
	stringMatcher(t, "secret user name", *secretOut.Username, userName)
	stringMatcher(t, "secret auth type", secretOut.AuthType, authType)
	stringMatcher(t, "secret name", secretOut.ResourceType, dao.SecretResourceType)

	encryptedPassword, err := util.Encrypt(password)
	if err != nil {
		t.Error(err)
	}

	stringMatcher(t, "secret password", *secretOut.Password, encryptedPassword)

	cleanSecretByID(t, secret.ID, &tenantId)
}

func stringMatcher(t *testing.T, nameResource, firstValue string, secondValue string) {
	if firstValue != secondValue {
		t.Errorf("Wrong %v, wanted %v got %v", nameResource, firstValue, secondValue)
	}
}

func int64Matcher(t *testing.T, nameResource string, firstValue int64, secondValue int64) {
	if firstValue != secondValue {
		t.Errorf("Wrong %v, wanted %v got %v", nameResource, firstValue, secondValue)
	}
}

func parseSecretResponse(t *testing.T, rec *httptest.ResponseRecorder) *m.AuthenticationResponse {
	secret := &m.AuthenticationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err := json.Unmarshal(raw, &secret)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	return secret
}

func fetchSecretFromDB(t *testing.T, secretIDValue string, secretTenantID int64) *m.Authentication {
	requestParams := dao.RequestParams{TenantID: &secretTenantID}
	secretDao := dao.GetSecretDao(&requestParams)
	secretID, err := util.InterfaceToInt64(secretIDValue)
	if err != nil {
		t.Error(err)
	}

	secret, err := secretDao.GetById(&secretID)
	if err != nil {
		t.Error(err)
	}

	return secret
}

func secretFromParams(secretName, secretAuthType, secretUserName, secretPassword string, secretExtra map[string]interface{}) *m.AuthenticationCreateRequest {
	return &m.AuthenticationCreateRequest{
		Name:     util.StringRef(secretName),
		AuthType: secretAuthType,
		Username: util.StringRef(secretUserName),
		Password: util.StringRef(secretPassword),
		Extra:    secretExtra,
	}
}

func createSecretRequest(t *testing.T, requestBody *m.AuthenticationCreateRequest, tenantIDValue *int64, userOwnership bool) (echo.Context, *httptest.ResponseRecorder, error) {
	requestInputBody := struct {
		*m.AuthenticationCreateRequest
		UserOwnership bool
	}{}

	requestInputBody.AuthenticationCreateRequest = requestBody
	requestInputBody.UserOwnership = userOwnership

	body, err := json.Marshal(requestInputBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/secrets",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": *tenantIDValue,
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	tenancy := middleware.Tenancy(SecretCreate)
	userCatcher := middleware.UserCatcher(tenancy)

	testUserId := "testUser"
	identityHeader := testutils.IdentityHeaderForUser(testUserId)
	c.Set("identity", identityHeader)

	err = userCatcher(c)

	return c, rec, err
}

func cleanSecretByID(t *testing.T, secretIDValue string, tenantId *int64) {
	secretID, err := util.InterfaceToInt64(secretIDValue)
	if err != nil {
		t.Error(err)
	}
	requestParams := dao.RequestParams{TenantID: tenantId}
	secretDao := dao.GetSecretDao(&requestParams)
	err = secretDao.Delete(&secretID)
	if err != nil {
		t.Error(err)
	}
}
