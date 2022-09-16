package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/middleware"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
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

	cleanSecretByID(t, secret.ID, &dao.RequestParams{TenantID: &tenantId})
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
	var userID *int64

	for _, userScoped := range []bool{false, true} {
		secretCreateRequest := secretFromParams(name, authType, userName, password, secretExtra, userScoped)

		c, rec, err := createSecretRequest(t, secretCreateRequest, &tenantId)
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

		if userScoped {
			secretUserID, ok := c.Get(h.USERID).(int64)
			if ok {
				userID = &secretUserID
			}
		}

		secretOut := fetchSecretFromDB(t, secret.ID, &dao.RequestParams{TenantID: &tenantId, UserID: userID})

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

		cleanSecretByID(t, secret.ID, &dao.RequestParams{TenantID: &tenantId, UserID: userID})
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

func fetchSecretFromDB(t *testing.T, secretIDValue string, requestParams *dao.RequestParams) *m.Authentication {
	secretDao := dao.GetSecretDao(requestParams)
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

func cleanSecretByID(t *testing.T, secretIDValue string, requestParams *dao.RequestParams) {
	secretID, err := util.InterfaceToInt64(secretIDValue)
	if err != nil {
		t.Error(err)
	}
	secretDao := dao.GetSecretDao(requestParams)
	err = secretDao.Delete(&secretID)
	if err != nil {
		t.Error(err)
	}
}

func TestSecretList(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantId := int64(1)
	secret1, err := dao.CreateSecretByName("Secret 1", &tenantId, nil)
	if err != nil {
		t.Error(err)
	}

	secret2, err := dao.CreateSecretByName("Secret 2", &tenantId, nil)
	if err != nil {
		t.Error(err)
	}

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err = SecretList(c)
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

	if out.Meta.Count != 2 {
		t.Error("not enough objects passed back from DB")
	}

	if len(out.Data) != 2 {
		t.Error("not enough objects passed back from DB")
	}

	_, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	var foundIDs []int64
	for _, secret := range out.Data {
		secretMap := secret.(map[string]interface{})
		secretID := secretMap["id"].(string)
		outID, err := strconv.ParseInt(secretID, 10, 64)
		if err != nil {
			t.Errorf(`The ID of the payload could not be converted to int64: %s`, err)
		}

		if outID == secret1.DbID || outID == secret2.DbID {
			foundIDs = append(foundIDs, outID)
		}
	}

	if len(foundIDs) != 2 {
		t.Errorf("Some secret IDs are missing, obtained: %v expected: %v", foundIDs, out.Data)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)

	secret1ID, err := util.InterfaceToString(secret1.DbID)
	if err != nil {
		t.Error(nil)
	}
	cleanSecretByID(t, secret1ID, &dao.RequestParams{TenantID: &tenantId})

	secret2ID, err := util.InterfaceToString(secret2.DbID)
	if err != nil {
		t.Error(nil)
	}
	cleanSecretByID(t, secret2ID, &dao.RequestParams{TenantID: &tenantId})
}

func TestSecretListWithFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantId := int64(1)
	secret1, err := dao.CreateSecretByName("Secret 1", &tenantId, nil)
	if err != nil {
		t.Error(err)
	}

	secret2, err := dao.CreateSecretByName("Secret 2", &tenantId, nil)
	if err != nil {
		t.Error(err)
	}

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "name", Value: []string{"Secret 2"}},
			},
			"tenantID": tenantId,
		},
	)

	err = SecretList(c)
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

	if out.Meta.Count != 1 {
		t.Error("not enough objects passed back from DB")
	}

	if len(out.Data) != 1 {
		t.Error("not enough objects passed back from DB")
	}

	_, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	var foundIDs []int64
	for _, secret := range out.Data {
		secretMap := secret.(map[string]interface{})
		secretID := secretMap["id"].(string)
		outID, err := strconv.ParseInt(secretID, 10, 64)
		if err != nil {
			t.Errorf(`The ID of the payload could not be converted to int64: %s`, err)
		}

		if outID == secret2.DbID {
			foundIDs = append(foundIDs, outID)
		}
	}

	if len(foundIDs) != 1 {
		t.Errorf("Some secret IDs are missing, obtained: %v expected: %v", foundIDs, out.Data)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)

	secret1ID, err := util.InterfaceToString(secret1.DbID)
	if err != nil {
		t.Error(nil)
	}
	cleanSecretByID(t, secret1ID, &dao.RequestParams{TenantID: &tenantId})

	secret2ID, err := util.InterfaceToString(secret2.DbID)
	if err != nil {
		t.Error(nil)
	}
	cleanSecretByID(t, secret2ID, &dao.RequestParams{TenantID: &tenantId})
}

func TestSecretTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantIDForSecret := int64(1)

	secret, err := dao.CreateSecretByName("Secret 1", &tenantIDForSecret, nil)
	if err != nil {
		t.Error(err)
	}

	tenantId := fixtures.NotExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	err = SecretList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)

	secretID, err := util.InterfaceToString(secret.DbID)
	if err != nil {
		t.Error(nil)
	}

	cleanSecretByID(t, secretID, &dao.RequestParams{TenantID: &tenantIDForSecret})
}

func TestSecretListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets",
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

	badRequestSecretList := ErrorHandlingContext(SecretList)
	err := badRequestSecretList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSecretGet(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantIDForSecret := int64(1)
	testUserId := "testUser"

	userDao := dao.GetUserDao(&tenantIDForSecret)
	user, err := userDao.FindOrCreate(testUserId)
	if err != nil {
		t.Error(err)
	}

	var userID *int64
	for _, userScoped := range []bool{false, true} {
		if userScoped {
			userID = &user.Id
		}

		secret, err := dao.CreateSecretByName("Secret 1", &tenantIDForSecret, userID)
		if err != nil {
			t.Error(err)
		}

		secretID, err := util.InterfaceToString(secret.DbID)
		if err != nil {
			t.Error(err)
		}

		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/secrets/"+secretID,
			nil,
			map[string]interface{}{
				"tenantID": tenantIDForSecret,
			},
		)

		c.SetParamNames("id")
		c.SetParamValues(secretID)

		userCatcher := middleware.UserCatcher(SecretGet)
		identityHeader := testutils.IdentityHeaderForUser(testUserId)
		c.Set("identity", identityHeader)

		err = userCatcher(c)
		if err != nil {
			t.Error(err)
		}

		if rec.Code != http.StatusOK {
			t.Error("Did not return 200")
		}

		var outSecret m.SecretResponse
		err = json.Unmarshal(rec.Body.Bytes(), &outSecret)
		if err != nil {
			t.Error("Failed unmarshaling output")
		}

		if outSecret.ID != secretID {
			t.Errorf(`wrong secret fetched. Want "%s", got "%s"`, secretID, outSecret.ID)
		}

		secretDao := dao.GetSecretDao(&dao.RequestParams{TenantID: &tenantIDForSecret, UserID: userID})
		secret, err = secretDao.GetById(&secret.DbID)
		if err != nil {
			t.Error("secret not found")
		}

		if userScoped && secret.UserID == nil || !userScoped && secret.UserID != nil {
			t.Error("user id has to be nil as user ownership was not requested for secret")
		}

		cleanSecretByID(t, secretID, &dao.RequestParams{TenantID: &tenantIDForSecret, UserID: userID})
	}
}

func TestSecretGetTenantNotExist(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	tenantID := fixtures.NotExistingTenantId
	tenantIDForSecret := int64(1)

	secret, err := dao.CreateSecretByName("Secret 1", &tenantIDForSecret, nil)
	if err != nil {
		t.Error(err)
	}

	secretID, err := util.InterfaceToString(secret.DbID)
	if err != nil {
		t.Error(err)
	}

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets/"+secretID,
		nil,
		map[string]interface{}{
			"tenantID": tenantID,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(secretID)

	notFoundSecretGet := ErrorHandlingContext(SecretGet)
	err = notFoundSecretGet(c)
	if err != nil {
		t.Error(err)
	}

	secretID, err = util.InterfaceToString(secret.DbID)
	if err != nil {
		t.Error(nil)
	}
	cleanSecretByID(t, secretID, &dao.RequestParams{TenantID: &tenantIDForSecret})

	templates.NotFoundTest(t, rec)
}

func TestSecretGetNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	id := "555555"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/secrets/"+id,
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	notFoundSecretGet := ErrorHandlingContext(SecretGet)
	err := notFoundSecretGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}
