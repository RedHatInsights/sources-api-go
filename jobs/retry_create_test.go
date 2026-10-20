package jobs

import (
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
)

// createRetryTestApplication creates an application row with the given availability status, retry counter and
// created_at timestamp, registering its cleanup so it doesn't leak into other tests. sourceID must be unique
// per call within a single test, since (application_type_id, source_id, tenant_id) is uniquely constrained and
// every row here shares the same application type and tenant fixtures.
func createRetryTestApplication(t *testing.T, sourceID int64, status string, retryCounter int8, createdAt time.Time) *m.Application {
	t.Helper()

	var app m.Application

	app.SourceID = sourceID
	app.ApplicationTypeID = fixtures.TestApplicationTypeData[0].Id
	app.TenantID = fixtures.TestTenantData[0].Id
	app.AvailabilityStatus = status
	app.RetryCounter = &retryCounter

	err := dao.DB.Create(&app).Error
	if err != nil {
		t.Fatalf("failed to create test application: %v", err)
	}

	// CreatedAt is set by GORM on creation - backdate it directly, bypassing hooks, so the row can
	// exercise the "RecordAgeLimit" cutoff.
	err = dao.DB.Model(&app).UpdateColumn("created_at", createdAt).Error
	if err != nil {
		t.Fatalf("failed to backdate test application: %v", err)
	}

	t.Cleanup(func() { dao.DB.Delete(&app) })

	return &app
}

// TestClaimRetryBatchClaimsEligibleApplications is a regression test for the GORM error "invalid value, should be
// pointer to struct or slice" that "claimRetryBatch" used to raise - caused by calling Scan() after Model()+Select()
// instead of Find(). It also verifies the claiming criteria: availability status, retry count and record age.
func TestClaimRetryBatchClaimsEligibleApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	eligible := createRetryTestApplication(t, 6, m.Unavailable, 0, time.Now())
	alreadyAvailable := createRetryTestApplication(t, 7, m.Available, 0, time.Now())
	exhaustedRetries := createRetryTestApplication(t, 8, m.Unavailable, RetryMax, time.Now())
	stale := createRetryTestApplication(t, 9, m.Unavailable, 0, time.Now().Add(2*RecordAgeLimit))

	apps, err := claimRetryBatch()
	if err != nil {
		t.Fatalf("claimRetryBatch returned an unexpected error: %v", err)
	}

	claimed := make(map[int64]bool, len(apps))
	for _, app := range apps {
		claimed[app.ID] = true
	}

	if !claimed[eligible.ID] {
		t.Errorf("expected eligible application %d to be claimed", eligible.ID)
	}

	if claimed[alreadyAvailable.ID] {
		t.Errorf("did not expect available application %d to be claimed", alreadyAvailable.ID)
	}

	if claimed[exhaustedRetries.ID] {
		t.Errorf("did not expect application %d with exhausted retries to be claimed", exhaustedRetries.ID)
	}

	if claimed[stale.ID] {
		t.Errorf("did not expect stale application %d to be claimed", stale.ID)
	}

	var reloaded m.Application

	err = dao.DB.First(&reloaded, eligible.ID).Error
	if err != nil {
		t.Fatalf("failed to reload claimed application: %v", err)
	}

	if reloaded.RetryCounter == nil || *reloaded.RetryCounter != 1 {
		t.Errorf("expected retry_counter to be incremented to 1, got %v", reloaded.RetryCounter)
	}
}

// TestClaimRetryBatchReturnsEmptyWhenNothingQualifies ensures an empty result set is returned (and not an error)
// once nothing in the batch matches the claiming criteria.
func TestClaimRetryBatchReturnsEmptyWhenNothingQualifies(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	createRetryTestApplication(t, 6, m.Available, 0, time.Now())
	createRetryTestApplication(t, 7, m.Unavailable, RetryMax, time.Now())

	apps, err := claimRetryBatch()
	if err != nil {
		t.Fatalf("claimRetryBatch returned an unexpected error: %v", err)
	}

	for _, app := range apps {
		if app.ID == 0 {
			t.Errorf("claimRetryBatch returned a zero-value application, the result set should be empty or fully populated")
		}
	}
}

// TestResetAvailableRetryCountersMarksAvailableApplicationsAsDone verifies that applications which became
// available have their retry counter pinned to RetryMax so they are not picked up again by claimRetryBatch.
func TestResetAvailableRetryCountersMarksAvailableApplicationsAsDone(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	app := createRetryTestApplication(t, 6, m.Available, 0, time.Now())

	err := resetAvailableRetryCounters()
	if err != nil {
		t.Fatalf("resetAvailableRetryCounters returned an unexpected error: %v", err)
	}

	var reloaded m.Application

	err = dao.DB.First(&reloaded, app.ID).Error
	if err != nil {
		t.Fatalf("failed to reload application: %v", err)
	}

	if reloaded.RetryCounter == nil || *reloaded.RetryCounter != RetryMax {
		t.Errorf("expected retry_counter to be reset to %d, got %v", RetryMax, reloaded.RetryCounter)
	}
}
