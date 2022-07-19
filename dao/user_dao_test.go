package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
)

func TestFindOrCreateUser(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("user_tests")

	userID := "test_user"

	tenantID := fixtures.TestTenantData[0].Id
	userDao := GetUserDao(&tenantID)

	user, err := userDao.FindOrCreate(userID)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	var expectedUser model.User
	err = DB.
		Debug().
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ?", user.Id, user.TenantID).
		First(&expectedUser).
		Error

	if err != nil {
		t.Errorf(`error fetching the tenant. Want nil error, got "%s"`, err)
	}

	want := userID
	got := expectedUser.UserID

	if want != got {
		t.Errorf(`unexpected user fetched - exptected user :%v ,obtained user:" %v`, want, got)
	}

	DropSchema("user_tests")
}
