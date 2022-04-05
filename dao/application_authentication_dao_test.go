package dao

import (
	"errors"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestDeleteApplicationAuthentication tests that an applicationAuthentication gets correctly deleted, and its data returned.
func TestDeleteApplicationAuthentication(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestSourceData[0].TenantID)

	applicationAuthentication := fixtures.TestApplicationAuthenticationData[0]
	// Set the ID to 0 to let GORM know it should insert a new applicationAuthentication and not update an existing one.
	applicationAuthentication.ID = 0
	// Set some data to compare the returned applicationAuthentication.
	applicationAuthentication.AuthenticationUID = "complex uuid"

	// Create the test applicationAuthentication.
	err := applicationAuthenticationDao.Create(&applicationAuthentication)
	if err != nil {
		t.Errorf("error creating applicationAuthentication: %s", err)
	}

	deletedApplicationAuthentication, err := applicationAuthenticationDao.Delete(&applicationAuthentication.ID)
	if err != nil {
		t.Errorf("error deleting an applicationAuthentication: %s", err)
	}

	{
		want := applicationAuthentication.ID
		got := deletedApplicationAuthentication.ID

		if want != got {
			t.Errorf(`incorrect applicationAuthentication deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	DoneWithFixtures("delete")
}

// TestDeleteApplicationAuthenticationNotExists tests that when an applicationAuthentication that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteApplicationAuthenticationNotExists(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := applicationAuthenticationDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DoneWithFixtures("delete")
}
