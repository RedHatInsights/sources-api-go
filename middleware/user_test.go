package middleware

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
)

var catchUserOrElse204 = UserCatcher(func(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
})

func TestUserCreationFromXRHID(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantID := int64(1)
	testUserID := "55555"
	identity := testutils.IdentityHeaderForUser(testUserID)

	database.ConnectAndMigrateDB("middleware")
	database.CreateFixtures("middleware")

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Set(h.TENANTID, tenantID)
	c.Set(h.PARSED_IDENTITY, identity)

	err := catchUserOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	var user m.User
	result := dao.DB.Find(&user, "user_id = ? AND tenant_id = ?", testUserID, tenantID)
	if result.Error != nil {
		t.Error(err)
	}

	if result.RowsAffected == 0 {
		t.Errorf("unable to find user %v", testUserID)
	}

	if user.Id == 0 || c.Get(h.USERID) != user.Id {
		t.Errorf("unable to find user id %v in context", user.Id)
	}

	database.DropSchema("middleware")
}

func TestUserCreationFromPSK(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantID := int64(1)
	testUserID := "55555"

	database.ConnectAndMigrateDB("middleware")
	database.CreateFixtures("middleware")

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Set(h.TENANTID, tenantID)
	c.Set(h.PSK_USER, testUserID)

	err := catchUserOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	var user m.User
	result := dao.DB.Find(&user, "user_id = ? AND tenant_id = ?", testUserID, tenantID)
	if result.Error != nil {
		t.Error(err)
	}

	if user.Id == 0 || c.Get(h.USERID) != user.Id {
		t.Errorf("unable to find user id %v in context", user.Id)
	}

	if result.RowsAffected == 0 {
		t.Errorf("unable to find user %v", testUserID)
	}

	database.DropSchema("middleware")
}
