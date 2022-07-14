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
	"github.com/RedHatInsights/sources-api-go/middleware"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
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

	err = cleanSourceForTenant(nameSource, &fixtures.TestTenantData[0].Id)
	if err != nil {
		t.Errorf(`unexpected error received when deleting the source: %s`, err)
	}

	err = dao.DB.Delete(&m.User{}, "user_id = ?", testUserId).Error
	if err != nil {
		t.Error(err)
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
