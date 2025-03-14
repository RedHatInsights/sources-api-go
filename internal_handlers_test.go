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

	if rec.Code != http.StatusOK {
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
		t.Errorf("not enough objects passed back from DB, want %d, got %d", len(fixtures.TestSourceData), len(out.Data))
	}

	for _, src := range out.Data {
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
		wantTenant := fixtures.TestTenantData[0]

		if wantTenant.ExternalTenant != responseExternalTenant {
			t.Errorf("Tenants don't match. Want %#v, got %#v", wantTenant.ExternalTenant, responseExternalTenant)
		}

		if wantTenant.OrgID != responseOrgId {
			t.Errorf("Org Ids don't match. Want %#v, got %#v", wantTenant.OrgID, responseOrgId)
		}

		sourceInFixtures := false
		for _, source := range fixtures.TestSourceData {
			if source.ID == responseSourceId {
				if source.AvailabilityStatus != responseAvailabilityStatus {
					t.Errorf("Availability statuses don't match. Want %s, got %s", source.AvailabilityStatus, responseAvailabilityStatus)
				}
				sourceInFixtures = true
				break
			}
		}
		if !sourceInFixtures {
			t.Errorf("Source ID %d not found in fixtures", responseSourceId)
		}
	}
	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestSourceListInternalSkipEmptySources tests that when the "skip empty sources" header is sent, the empty sources
// and the ones that just have Cost Management applications are skipped.
func TestSourceListInternalSkipEmptySources(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	requestTester := func(sourceIDWithoutApplications int64, expectedNumberElements int, shouldFindSourceWithoutApplications bool) {
		// Create the fake request.
		context, recorder := request.CreateTestContext(http.MethodGet, "/internal/v2.0/sources", nil, map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []util.Filter{},
			"headers": map[string]string{h.SkipEmptySources: "true"},
		})

		// Call the handler.
		err := InternalSourceList(context)
		if err != nil {
			t.Error(err)
		}

		// Make sure we get an OK response.
		if recorder.Code != http.StatusOK {
			t.Error("Did not return 200")
		}

		var out util.Collection
		err = json.Unmarshal(recorder.Body.Bytes(), &out)
		if err != nil {
			t.Error("Failed unmarshalling output")
		}

		// Assert that an expected number of elements was returned.
		if len(out.Data) != expectedNumberElements {
			t.Errorf(`unexpected number of sources returned, want "%d", got "%d"`, expectedNumberElements, len(out.Data))
		}

		// Assert that the source with the Cost Management applications is or isn't present in the results.
		var found = false
		for _, src := range out.Data {
			s, ok := src.(map[string]interface{})
			if !ok {
				t.Error("model did not deserialize as a source")
			}

			// Parse the source
			responseSourceId, err := util.InterfaceToInt64(s["id"])
			if err != nil {
				t.Errorf("could not parse id from response: %s", err)
			}

			if responseSourceId == sourceIDWithoutApplications {
				if shouldFindSourceWithoutApplications {
					found = true
					break
				} else {
					t.Errorf("the source with just a Cost Management application is in the response, when it shouldn't")
				}
			}
		}

		if shouldFindSourceWithoutApplications && !found {
			t.Errorf("the source with just a Cost Management application should have been returned, because now it has an associated RHC Connection with it")
		}
	}

	// Create a "Cost Management" application for the source without applications, which should not be returned in the
	// response.
	sourceWithoutApplications := fixtures.TestSourceData[4] // Source without applications.
	application := &m.Application{
		ApplicationTypeID:  fixtures.TestApplicationTypeData[5].Id, // "Cost Management" application type.
		AvailabilityStatus: "available",
		SourceID:           sourceWithoutApplications.ID,
		TenantID:           1,
	}

	applicationDao := dao.GetApplicationDao(&dao.RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	if err := applicationDao.Create(application); err != nil {
		t.Errorf("unable to create Cost Management application: %s", err)
	}

	// Clean up the created application.
	defer func() {
		_, err := applicationDao.Delete(&application.ID)
		if err != nil {
			t.Errorf("unable to clean up the created application. Expect many other tests to fail: %s", err)
		}
	}()

	requestTester(sourceWithoutApplications.ID, 3, false)

	// Associate an RHC Connection with the "applicationless source".
	rhcConnection := &m.RhcConnection{
		ID:                 12345,
		RhcId:              "abcde",
		Sources:            []m.Source{sourceWithoutApplications},
		AvailabilityStatus: "available",
	}

	rhcConnectionDao := dao.GetRhcConnectionDao(&dao.RequestParams{TenantID: &fixtures.TestTenantData[0].Id})
	if _, err := rhcConnectionDao.Create(rhcConnection); err != nil {
		t.Errorf("unable to create the RHC Connection: %s", err)
	}

	// Clean up the created RHC Connection.
	defer func() {
		_, err := rhcConnectionDao.Delete(&rhcConnection.ID)
		if err != nil {
			t.Errorf("unable to clean up the created RHC Connection. Expect many other tests to fail: %s", err)
		}
	}()

	requestTester(sourceWithoutApplications.ID, 4, true)
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
			c.Set(h.UserID, user.Id)
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
