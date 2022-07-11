package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/RedHatInsights/sources-api-go/middleware"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func TestBulkCreateMissingSourceType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	nameSource := "test"
	bulkCreateSource := m.BulkCreateSource{SourceCreateRequest: m.SourceCreateRequest{Name: &nameSource}}
	requestBody := m.BulkCreateRequest{Sources: []m.BulkCreateSource{bulkCreateSource}}
	testUserId := "testUser"

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulkc_reate",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

	user, err := dao.GetUserDao(&fixtures.TestTenantData[0].Id).FindOrCreate(testUserId)
	if err != nil {
		t.Error(err)
	}

	c.Set(h.USERID, user.Id)

	err = BulkCreate(c)
	if err.Error() != "no source type present, need either [source_type_name] or [source_type_id]" {
		t.Error(err)
	}

	err = dao.DB.Delete(&m.User{}, "user_id = ?", testUserId).Error
	if err != nil {
		t.Error(err)
	}
}

func TestBulkCreateWithUserCreation(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	conf.ResourceOwnership = "user"

	testUserId := "testUser"
	identityHeader := testutils.IdentityHeaderForUser(testUserId)

	nameSource := "test source"
	sourceTypeName := "bitbucket"
	applicationTypeName := "app-studio"
	authenticationResourceType := "application"

	requestBody := testutils.SingleResourceBulkCreateRequest(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType)

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, res := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulk_create",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", identityHeader)

	var user m.User
	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).First(&user).Error
	if err.Error() != "record not found" {
		t.Error(err)
	}

	bulkCreate := middleware.UserCatcher(BulkCreate)
	err = bulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).First(&user).Error
	if err != nil {
		t.Error(err)
	}

	if user.UserID != testUserId {
		t.Errorf("expected userid is %s instead of %s", testUserId, user.UserID)
	}

	var source m.Source
	err = dao.DB.Model(&m.Source{}).Where("name = ?", nameSource).First(&source).Error
	if err != nil {
		t.Error(err)
	}

	if source.UserID == nil || *source.UserID != user.Id {
		t.Error("source user id was not populated correctly")
	}

	var response m.BulkCreateResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	if err != nil {
		t.Error(err)
	}

	var application m.Application
	err = dao.DB.Model(&m.Application{}).Where("id = ?", response.Applications[0].ID).First(&application).Error
	if err != nil {
		t.Error(err)
	}

	if application.UserID == nil || *application.UserID != user.Id {
		t.Error("application user id was not populated correctly")
	}

	var authentication m.Authentication
	err = dao.DB.Model(&m.Authentication{}).Where("id = ?", response.Authentications[0].ID).First(&authentication).Error
	if err != nil {
		t.Error(err)
	}

	if authentication.UserID == nil || *authentication.UserID != user.Id {
		t.Error("authentication user id was not populated correctly")
	}

	var applicationAuthentication m.ApplicationAuthentication
	err = dao.DB.Model(&m.ApplicationAuthentication{}).
		Where("application_id = ? AND authentication_id = ?", response.Applications[0].ID, response.Authentications[0].ID).
		Find(&applicationAuthentication).Error
	if err != nil {
		t.Error(err)
	}

	if applicationAuthentication.UserID == nil || *applicationAuthentication.UserID != user.Id {
		t.Error(err)
	}

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}

	err = dao.DB.Delete(&m.User{}, "user_id = ?", testUserId).Error
	if err != nil {
		t.Error(err)
	}
}

func TestBulkCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	conf.ResourceOwnership = ""

	testUserId := "testUser"
	identityHeader := testutils.IdentityHeaderForUser(testUserId)

	nameSource := "test source"
	sourceTypeName := "bitbucket"
	applicationTypeName := "app-studio"
	authenticationResourceType := "application"

	requestBody := testutils.SingleResourceBulkCreateRequest(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType)

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, res := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulk_create",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", identityHeader)

	err = BulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	var response m.BulkCreateResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	if err != nil {
		t.Error(err)
	}

	source := response.Sources[0]
	if *source.Name != nameSource {
		t.Errorf("expected source: %v, got %v", nameSource, source.Name)
	}

	application := response.Applications[0]
	if application.SourceID != source.ID {
		t.Errorf("expected source id in application: %v, got %v", source.ID, application.SourceID)
	}

	endpoint := response.Endpoints[0]
	if endpoint.SourceID != source.ID {
		t.Errorf("expected source id in endpoint: %v, got %v", source.ID, endpoint.SourceID)
	}

	authentication := response.Authentications[0]
	if authentication.ResourceID != application.ID {
		t.Errorf("expected resource id in authentication: %v, got %v", application.ID, authentication.ResourceID)
	}

	if authentication.ResourceType != "Application" {
		t.Errorf("expected resource type in authentication: Application, got %v", authentication.ResourceType)
	}

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}
}

func TestBulkCreateSourceValidationBadRequest(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(1)

	// Create a source
	reqParams := dao.RequestParams{TenantID: &tenantId}
	sourceDao := dao.GetSourceDao(&reqParams)

	uid := "bd2ba6d6-4630-40e2-b829-cf09b03bdb9f"
	nameSource := "Source for TestBulkCreateSourceValidationBadRequest()"
	src := m.Source{
		Name:         nameSource,
		SourceTypeID: 1,
		Uid:          &uid,
	}

	err := sourceDao.Create(&src)
	if err != nil {
		t.Errorf("source not created correctly: %s", err)
	}

	// Try to create same source via bulk create
	sourceTypeName := "bitbucket"
	applicationTypeName := "app-studio"
	authenticationResourceType := "application"

	requestBody := testutils.SingleResourceBulkCreateRequest(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType)

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, res := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulk_create",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

	badRequestBulkCreate := ErrorHandlingContext(BulkCreate)
	err = badRequestBulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, res)

	// Delete created source
	deletedSource, err := sourceDao.Delete(&src.ID)
	if err != nil {
		t.Error(err)
	}
	if deletedSource.ID != src.ID {
		t.Error("wrong source deleted")
	}
}

func cleanSourceForTenant(sourceName string, tenantID *int64) error {
	source := &m.Source{Name: sourceName}
	err := dao.DB.Model(&m.Source{}).Where("name = ?", source.Name).Find(&source).Error
	if err != nil {
		return err
	}

	err = service.DeleteCascade(tenantID, nil, "Source", source.ID, []kafka.Header{})

	return err
}
