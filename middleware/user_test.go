package middleware

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v5"
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

	c.Set(h.TenantID, tenantID)
	c.Set(h.ParsedIdentity, identity)

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

	if user.Id == 0 || c.Get(h.UserID) != user.Id {
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

	c.Set(h.TenantID, tenantID)
	c.Set(h.PSKUserID, testUserID)

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

	if user.Id == 0 || c.Get(h.UserID) != user.Id {
		t.Errorf("unable to find user id %v in context", user.Id)
	}

	if result.RowsAffected == 0 {
		t.Errorf("unable to find user %v", testUserID)
	}

	database.DropSchema("middleware")
}

// TestEmptyIncomingUserIdNoopMiddleware tests that when no "user ID" is
// present in the "Identity", the middleware does not return an error or does
// not set the user ID. It's a regression test for RHCLOUD-42337.
func TestEmptyIncomingUserIdNoopMiddleware(t *testing.T) {
	// Create an identity header and remove the user from it.
	identity := testutils.IdentityHeaderForUser("12345")
	identity.Identity.User = nil

	// Create the request and set up the proper headers.
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	c.Set(h.TenantID, fixtures.TestTenantData[0].Id)
	c.Set(h.ParsedIdentity, identity)

	// Call the middleware under test.
	err := catchUserOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`unexpected status code received. Want "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Make sure no user was set in the context.
	userId := c.Get(h.UserID)
	if userId != nil {
		t.Errorf("want no user ID set in the context, got following user ID: %v", userId)
	}
}
