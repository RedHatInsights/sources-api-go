package dao

import (
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
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
