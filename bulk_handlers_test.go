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
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func TestBulkCreateMissingSourceType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	nameSource := "test"
	bulkCreateSource := m.BulkCreateSource{SourceCreateRequest: m.SourceCreateRequest{Name: &nameSource}}
	requestBody := m.BulkCreateRequest{Sources: []m.BulkCreateSource{bulkCreateSource}}

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

	err = BulkCreate(c)
	if err.Error() != "no source type present, need either [source_type_name] or [source_type_id]" {
		t.Error(err)
	}
}

func TestCreateUserWithResourceOwnershipApplicationType(t *testing.T) {
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
	c.Set("identity", &identity.XRHID{Identity: identityHeader})

	err = BulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	var users []m.User
	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).Find(&users).Error
	if err != nil {
		t.Error(err)
	}

	if len(users) != 1 {
		t.Errorf("1 user expected instead of %d", len(users))
	}

	if users[0].UserID != testUserId {
		t.Errorf("expected userid is %s instead of %s", testUserId, users[0].UserID)
	}

	var sources []m.Source
	err = dao.DB.Model(&m.Source{}).Where("name = ?", nameSource).Find(&sources).Error
	if err != nil {
		t.Error(err)
	}

	if *(sources[0].UserID) != users[0].Id {
		t.Error(err)
	}

	var response m.BulkCreateResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	if err != nil {
		t.Error(err)
	}

	var applications []m.Application
	err = dao.DB.Model(&m.Application{}).Where("id = ?", response.Applications[0].ID).Find(&applications).Error
	if err != nil {
		t.Error(err)
	}

	if *(applications[0].UserID) != users[0].Id {
		t.Error(err)
	}

	var authentications []m.Authentication
	err = dao.DB.Model(&m.Authentication{}).Where("id = ?", response.Authentications[0].ID).Find(&authentications).Error
	if err != nil {
		t.Error(err)
	}

	if *(authentications[0].UserID) != users[0].Id {
		t.Error(err)
	}

	var applicationAuthentications []m.ApplicationAuthentication
	err = dao.DB.Model(&m.ApplicationAuthentication{}).
		Where("application_id = ? AND authentication_id = ?", response.Applications[0].ID, response.Authentications[0].ID).
		Find(&applicationAuthentications).Error
	if err != nil {
		t.Error(err)
	}

	if *(authentications[0].UserID) != users[0].Id {
		t.Error(err)
	}

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}

	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).Delete(&users).Error
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUserWithoutResourceOwnershipApplicationType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	conf.ResourceOwnership = "user"

	testUserId := "testUser"
	identityHeader := testutils.IdentityHeaderForUser(testUserId)

	nameSource := "test source"
	sourceTypeName := "amazon"
	applicationTypeName := "cost-management"
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
	c.Set("identity", &identity.XRHID{Identity: identityHeader})

	err = BulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	var users []m.User
	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).Find(&users).Error
	if err != nil {
		t.Error(err)
	}

	if len(users) != 0 {
		t.Errorf("0 user expected instead of %d", len(users))
	}

	var sources []m.Source
	err = dao.DB.Model(&m.Source{}).Where("name = ?", nameSource).Find(&sources).Error
	if err != nil {
		t.Error(err)
	}

	if sources[0].UserID != nil {
		t.Error(err)
	}

	var response m.BulkCreateResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	if err != nil {
		t.Error(err)
	}

	var applications []m.Application
	err = dao.DB.Model(&m.Application{}).Where("id = ?", response.Applications[0].ID).Find(&applications).Error
	if err != nil {
		t.Error(err)
	}

	if applications[0].UserID != nil {
		t.Error(err)
	}

	var authentications []m.Authentication
	err = dao.DB.Model(&m.Authentication{}).Where("id = ?", response.Authentications[0].ID).Find(&authentications).Error
	if err != nil {
		t.Error(err)
	}

	if authentications[0].UserID != nil {
		t.Error(err)
	}

	var applicationAuthentications []m.ApplicationAuthentication
	err = dao.DB.Model(&m.ApplicationAuthentication{}).
		Where("application_id = ? AND authentication_id = ?", response.Applications[0].ID, response.Authentications[0].ID).
		Find(&applicationAuthentications).Error

	if err != nil {
		t.Error(err)
	}

	if authentications[0].UserID != nil {
		t.Error(err)
	}

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}
}

func TestCreateUserWithoutResourceOwnershipConfig(t *testing.T) {
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

	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulk_create",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identityHeader})

	err = BulkCreate(c)
	if err != nil {
		t.Error(err)
	}

	var users []m.User
	err = dao.DB.Model(&m.User{}).Where("user_id = ?", testUserId).Find(&users).Error
	if err != nil {
		t.Error(err)
	}

	if len(users) != 0 {
		t.Errorf("0 users expected instead of %d", len(users))
	}

	var sources []m.Source
	err = dao.DB.Model(&m.Source{}).Where("name = ?", nameSource).Find(&sources).Error
	if err != nil {
		t.Error(err)
	}

	if sources[0].UserID != nil {
		t.Error(err)
	}

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}
}

func cleanSourceForTenant(sourceName string, tenantID *int64) error {
	sourceDao := dao.GetSourceDao(&dao.SourceDaoParams{TenantID: tenantID})

	source := &m.Source{Name: sourceName}
	err := dao.DB.Model(&m.Source{}).Where("name = ?", source.Name).Find(&source).Error
	if err != nil {
		return err
	}

	_, _, _, _, _, err = sourceDao.DeleteCascade(source.ID)

	return err
}
