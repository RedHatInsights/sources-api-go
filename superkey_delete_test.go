package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/jobs"
	"github.com/RedHatInsights/sources-api-go/middleware"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// mockEnqueuedJobs collects jobs that would have been enqueued during a test.
var mockEnqueuedJobs []jobs.Job

// mockEnqueue replaces jobs.Enqueue in tests to avoid needing a running
// valkey instance. It records the enqueued job for later assertions.
func mockEnqueue(j jobs.Job) {
	mockEnqueuedJobs = append(mockEnqueuedJobs, j)
}

// setupSuperKeyTest swaps the DAO factory and Enqueue function for testing.
// It returns a cleanup function that restores the original values.
func setupSuperKeyTest(superKeyEnabled bool) func() {
	// Save originals
	origGetSourceDao := dao.GetSourceDao
	origGetApplicationDao := dao.GetApplicationDao
	origEnqueue := jobs.Enqueue

	// Set up mock DAOs with configurable IsSuperkey
	dao.GetSourceDao = func(_ *dao.RequestParams) dao.SourceDao {
		return &mocks.MockSourceDao{
			Sources:         fixtures.TestSourceData,
			RelatedSources:  fixtures.TestSourceData,
			SuperKeyEnabled: superKeyEnabled,
		}
	}

	dao.GetApplicationDao = func(_ *dao.RequestParams) dao.ApplicationDao {
		return &mocks.MockApplicationDao{
			Applications:    fixtures.TestApplicationData,
			SuperKeyEnabled: superKeyEnabled,
		}
	}

	// Mock Enqueue
	mockEnqueuedJobs = nil
	jobs.Enqueue = mockEnqueue

	return func() {
		dao.GetSourceDao = origGetSourceDao
		dao.GetApplicationDao = origGetApplicationDao
		jobs.Enqueue = origEnqueue
	}
}

// buildXRHIdentity creates a base64-encoded x-rh-identity header value for tests.
func buildXRHIdentity() string {
	id := identity.XRHID{Identity: identity.Identity{AccountNumber: "12345", OrgID: "23456"}}
	raw, _ := json.Marshal(id)

	return string(base64.StdEncoding.EncodeToString(raw))
}

// TestSuperKeyDestroyApplicationReturns202 verifies that a DELETE request
// for a superkey application is intercepted by the middleware and returns
// 202 Accepted (async delete) instead of the immediate 204 No Content.
func TestSuperKeyDestroyApplicationReturns202(t *testing.T) {
	cleanup := setupSuperKeyTest(true)
	defer cleanup()

	appId := fixtures.TestApplicationData[0].ID
	id := fmt.Sprintf("%d", appId)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/"+id,
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
			h.XRHID:    buildXRHIdentity(),
			// ForwadableHeaders reads these from context
			"x-rh-sources-account-number": "12345",
			"x-rh-sources-org-id":         "23456",
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	// Wrap the handler with the superkey destroy middleware, just like routes.go does
	handler := middleware.SuperKeyDestroyApplication(ApplicationDelete)
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d (Accepted), got %d", http.StatusAccepted, rec.Code)
	}

	if len(mockEnqueuedJobs) != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", len(mockEnqueuedJobs))
	}

	job, ok := mockEnqueuedJobs[0].(*jobs.SuperkeyDestroyJob)
	if !ok {
		t.Fatalf("expected SuperkeyDestroyJob, got %T", mockEnqueuedJobs[0])
	}

	if job.Model != "application" {
		t.Errorf("expected job model 'application', got %q", job.Model)
	}

	if job.Id != appId {
		t.Errorf("expected job id %d, got %d", appId, job.Id)
	}

	if job.Tenant != tenantId {
		t.Errorf("expected job tenant %d, got %d", tenantId, job.Tenant)
	}
}

// TestSuperKeyDestroySourceReturns202 verifies that a DELETE request for a
// superkey source is intercepted and returns 202 Accepted.
func TestSuperKeyDestroySourceReturns202(t *testing.T) {
	cleanup := setupSuperKeyTest(true)
	defer cleanup()

	sourceId := fixtures.TestSourceData[0].ID
	id := fmt.Sprintf("%d", sourceId)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/"+id,
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
			h.XRHID:    buildXRHIdentity(),
			"x-rh-sources-account-number": "12345",
			"x-rh-sources-org-id":         "23456",
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	handler := middleware.SuperKeyDestroySource(SourceDelete)
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d (Accepted), got %d", http.StatusAccepted, rec.Code)
	}

	if len(mockEnqueuedJobs) != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", len(mockEnqueuedJobs))
	}

	job, ok := mockEnqueuedJobs[0].(*jobs.SuperkeyDestroyJob)
	if !ok {
		t.Fatalf("expected SuperkeyDestroyJob, got %T", mockEnqueuedJobs[0])
	}

	if job.Model != "source" {
		t.Errorf("expected job model 'source', got %q", job.Model)
	}

	if job.Id != sourceId {
		t.Errorf("expected job id %d, got %d", sourceId, job.Id)
	}
}

// TestNonSuperKeyApplicationDeleteReturns204 verifies that a DELETE request
// for a non-superkey application passes through the middleware and reaches
// the handler, which returns 204 No Content.
func TestNonSuperKeyApplicationDeleteReturns204(t *testing.T) {
	cleanup := setupSuperKeyTest(false)
	defer cleanup()

	appId := fixtures.TestApplicationData[0].ID
	id := fmt.Sprintf("%d", appId)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/"+id,
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	handler := middleware.SuperKeyDestroyApplication(ApplicationDelete)
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d (NoContent), got %d", http.StatusNoContent, rec.Code)
	}

	if len(mockEnqueuedJobs) != 0 {
		t.Errorf("expected no enqueued jobs for non-superkey delete, got %d", len(mockEnqueuedJobs))
	}
}

// TestNonSuperKeySourceDeleteReturns204 verifies that a DELETE request for a
// non-superkey source passes through the middleware to the handler (204).
func TestNonSuperKeySourceDeleteReturns204(t *testing.T) {
	cleanup := setupSuperKeyTest(false)
	defer cleanup()

	sourceId := fixtures.TestSourceData[0].ID
	id := fmt.Sprintf("%d", sourceId)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/"+id,
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	handler := middleware.SuperKeyDestroySource(SourceDelete)
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d (NoContent), got %d", http.StatusNoContent, rec.Code)
	}

	if len(mockEnqueuedJobs) != 0 {
		t.Errorf("expected no enqueued jobs for non-superkey delete, got %d", len(mockEnqueuedJobs))
	}
}

// TestSuperKeyDestroyApplicationJobContents verifies that the enqueued
// SuperkeyDestroyJob contains the correct identity and forwarded headers.
func TestSuperKeyDestroyApplicationJobContents(t *testing.T) {
	cleanup := setupSuperKeyTest(true)
	defer cleanup()

	appId := fixtures.TestApplicationData[0].ID
	id := fmt.Sprintf("%d", appId)
	tenantId := int64(1)
	xrhid := buildXRHIdentity()

	c, _ := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/"+id,
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
			h.XRHID:    xrhid,
			"x-rh-sources-account-number": "12345",
			"x-rh-sources-org-id":         "23456",
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(id)

	handler := middleware.SuperKeyDestroyApplication(ApplicationDelete)
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockEnqueuedJobs) != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", len(mockEnqueuedJobs))
	}

	job := mockEnqueuedJobs[0].(*jobs.SuperkeyDestroyJob)

	if job.Identity != xrhid {
		t.Errorf("expected job identity to match x-rh-identity header")
	}

	if job.Tenant != tenantId {
		t.Errorf("expected job tenant %d, got %d", tenantId, job.Tenant)
	}
}

// TestSuperKeyDestroyApplicationBadId verifies that a non-numeric id
// returns an error from the middleware.
func TestSuperKeyDestroyApplicationBadId(t *testing.T) {
	cleanup := setupSuperKeyTest(true)
	defer cleanup()

	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/notanumber",
		nil,
		map[string]interface{}{
			h.TenantID: tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("notanumber")

	handler := ErrorHandlingContext(middleware.SuperKeyDestroyApplication(ApplicationDelete))
	err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d (BadRequest), got %d", http.StatusBadRequest, rec.Code)
	}
}
