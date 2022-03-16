package dao

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
)

// testApplication holds a test application in order to avoid having to write the "fixtures..." stuff every time.
var testApplication = fixtures.TestApplicationData[0]

// TestPausingApplication tests that an application gets correctly paused when using the method from the DAO.
func TestPausingApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	CreateFixtures("pause_unpause")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)
	err := applicationDao.Pause(testApplication.ID)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	application, err := applicationDao.GetById(&testApplication.ID)
	if err != nil {
		t.Errorf(`error fetching the application. Want nil error, got "%s"`, err)
	}

	want := time.Now()
	if !dateTimesAreSimilar(want, application.PausedAt) {
		t.Errorf(`want now, got "%s"`, application.Pause.PausedAt)
	}

	DoneWithFixtures("pause_unpause")
}

// TestResumeApplication tests that the application is properly resumed when using the method from the DAO.
func TestResumeApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	CreateFixtures("pause_unpause")

	applicationDao := GetApplicationDao(&testApplication.TenantID)
	err := applicationDao.Unpause(testApplication.ID)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	application, err := applicationDao.GetById(&testApplication.ID)
	if err != nil {
		t.Errorf(`error fetching the application. Want nil error, got "%s"`, err)
	}

	var want time.Time
	if want != application.PausedAt {
		t.Errorf(`want "%s", got "%s"`, want, application.Pause.PausedAt)
	}

	DoneWithFixtures("pause_unpause")
}

// TestDeleteApplication tests that an application gets correctly deleted, and its data returned.
func TestDeleteApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)

	application := fixtures.TestApplicationData[0]
	// Set the ID to 0 to let GORM know it should insert a new application and not update an existing one.
	application.ID = 0
	// Set some data to compare the returned application.
	application.Extra = []byte(`{"hello": "world"}`)

	// Create the test application.
	err := applicationDao.Create(&application)
	if err != nil {
		t.Errorf("error creating application: %s", err)
	}

	deletedApplication, err := applicationDao.Delete(&application.ID)
	if err != nil {
		t.Errorf("error deleting an application: %s", err)
	}

	{
		want := application.ID
		got := deletedApplication.ID

		if want != got {
			t.Errorf(`incorrect application deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	{
		want := application.Extra
		got := deletedApplication.Extra

		if !bytes.Equal(want, got) {
			t.Errorf(`incorrect application deleted. Want "%s" in the extra field, got "%s"`, want, got)
		}
	}

	DoneWithFixtures("delete")
}

// TestDeleteApplicationNotExists tests that when an application that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteApplicationNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	applicationDao := GetApplicationDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := applicationDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DoneWithFixtures("delete")
}
