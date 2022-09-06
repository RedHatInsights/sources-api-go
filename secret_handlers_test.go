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

const secretResourceType = "Tenant"

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

	secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra, false)
	_, rec, err := createSecretRequest(t, secretCreateRequest, &tenantId)
	if err != nil {
		t.Error(err)
	}

	secret := parseSecretResponse(t, rec)

	_, _, err = createSecretRequest(t, secretCreateRequest, &tenantId)

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

	secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra, false)

	_, _, err := createSecretRequest(t, secretCreateRequest, &tenantId)

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

	for _, userScoped := range []bool{false, true} {
		secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra, userScoped)

		_, rec, err := createSecretRequest(t, secretCreateRequest, &tenantId)
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
		stringMatcher(t, "secret name", secretOut.ResourceType, secretResourceType)

		if userScoped && secretOut.UserID == nil || !userScoped && secretOut.UserID != nil {
			t.Error("user id has to be nil as user ownership was not requested for secret")
		}

		encryptedPassword, err := util.Encrypt(password)
		if err != nil {
			t.Error(err)
		}

		stringMatcher(t, "secret password", *secretOut.Password, encryptedPassword)

		cleanSecretByID(t, secret.ID, &tenantId)
	}
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

func secretFromParams(secretName, secretAuthType, secretUserName, secretPassword string, secretExtra map[string]interface{}, userScoped bool) *m.SecretCreateRequest {
	secretUserScoped := false
	if userScoped {
		secretUserScoped = true
	}

	return &m.SecretCreateRequest{
		Name:       util.StringRef(secretName),
		AuthType:   secretAuthType,
		Username:   util.StringRef(secretUserName),
		Password:   util.StringRef(secretPassword),
		Extra:      secretExtra,
		UserScoped: secretUserScoped,
	}
}

func createSecretRequest(t *testing.T, requestBody *m.SecretCreateRequest, tenantIDValue *int64) (echo.Context, *httptest.ResponseRecorder, error) {
	body, err := json.Marshal(requestBody)
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
