package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceListInternal(t *testing.T) {
	t.Skip("Skipping test (it needs to match results correctly)")

	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/internal/v2.0/sources",
		nil,
		map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []util.Filter{},
		})

	err := InternalSourceList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
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

	if len(out.Data) != len(fixtures.TestSourceData) {
		t.Error("not enough objects passed back from DB")
	}

	for i, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		// Parse the source
		responseSourceId, err := util.InterfaceToInt64(s["id"])
		if err != nil {
			t.Errorf("could not parse id from response: %s", err)
		}

		responseExternalTenant := s["tenant"].(string)
		responseOrgId := s["org_id"].(string)
		responseAvailabilityStatus := s["availability_status"].(string)

		// Check that the expected source data and the received data are the same
		if want := fixtures.TestSourceData[i].ID; want != responseSourceId {
			t.Errorf("Ids don't match. Want %d, got %d", want, responseSourceId)
		}

		if want := fixtures.TestTenantData[0].ExternalTenant; want != responseExternalTenant {
			t.Errorf("Tenants don't match. Want %#v, got %#v", want, responseExternalTenant)
		}

		if want := fixtures.TestTenantData[0].OrgID; want != responseOrgId {
			t.Errorf("Org Ids don't match. Want %#v, got %#v", want, responseOrgId)
		}

		if want := fixtures.TestSourceData[i].AvailabilityStatus; want != responseAvailabilityStatus {
			t.Errorf("Availability statuses don't match. Want %s, got %s", want, responseAvailabilityStatus)
		}
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListInternalBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/internal/v2.0/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
		},
	)

	badRequestInternalSourceList := ErrorHandlingContext(InternalSourceList)
	err := badRequestInternalSourceList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestInternalSecretGet(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	tenantIDForSecret := int64(1)
	testUserId := "testUser"

	userDao := dao.GetUserDao(&tenantIDForSecret)
	user, err := userDao.FindOrCreate(testUserId)
	if err != nil {
		t.Error(err)
	}

	encryptedPassword, err := util.Encrypt("password")
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
			"/api/internal/v2.0/secrets/"+secretID,
			nil,
			map[string]interface{}{
				"tenantID": tenantIDForSecret,
			},
		)

		c.SetParamNames("id")
		c.SetParamValues(secretID)

		if userID != nil {
			c.Set(h.USERID, user.Id)
		}

		err = InternalSecretGet(c)
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

		if *secret.Password != encryptedPassword {
			t.Errorf("expected password %v but got %v", encryptedPassword, *secret.Password)
		}

		if userScoped && secret.UserID == nil || !userScoped && secret.UserID != nil {
			t.Error("user id has to be nil as user ownership was not requested for secret")
		}

		cleanSecretByID(t, secretID, &dao.RequestParams{TenantID: &tenantIDForSecret, UserID: userID})
	}
}
