package model

import (
	"os"
	"reflect"
	"testing"
	"time"

	"gorm.io/datatypes"
)

func TestToResponseWithDependentApplications(t *testing.T) {
	now := time.Now()
	at := ApplicationType{
		Id:                    1,
		CreatedAt:             now,
		UpdatedAt:             now,
		Name:                  "/insights/platform/cost-management",
		DisplayName:           "Cost Management",
		DependentApplications: datatypes.JSON(`["app1","app2","app3"]`),
		SupportedSourceTypes:  datatypes.JSON(`["amazon"]`),
	}

	resp := at.ToResponse()

	if resp.Id != "1" {
		t.Errorf("expected id '1', got '%s'", resp.Id)
	}

	if resp.Name != "/insights/platform/cost-management" {
		t.Errorf("expected name '/insights/platform/cost-management', got '%s'", resp.Name)
	}

	if resp.DisplayName != "Cost Management" {
		t.Errorf("expected display name 'Cost Management', got '%s'", resp.DisplayName)
	}

	expected := []string{"app1", "app2", "app3"}
	if !reflect.DeepEqual(resp.DependentApplications, expected) {
		t.Errorf("expected dependent_applications %v, got %v", expected, resp.DependentApplications)
	}
}

func TestToResponseWithEmptyDependentApplications(t *testing.T) {
	at := ApplicationType{
		Id:                    2,
		Name:                  "/insights/platform/test",
		DisplayName:           "Test",
		DependentApplications: datatypes.JSON(`[]`),
	}

	resp := at.ToResponse()

	if len(resp.DependentApplications) != 0 {
		t.Errorf("expected empty dependent_applications, got %v", resp.DependentApplications)
	}

	// Ensure it's an initialized empty slice, not nil (so JSON serializes to [] not null)
	if resp.DependentApplications == nil {
		t.Error("expected non-nil empty slice for dependent_applications, got nil")
	}
}

func TestToResponseWithNilDependentApplications(t *testing.T) {
	at := ApplicationType{
		Id:                    3,
		Name:                  "/insights/platform/nil-test",
		DisplayName:           "Nil Test",
		DependentApplications: nil,
	}

	resp := at.ToResponse()

	if resp.DependentApplications == nil {
		t.Error("expected non-nil empty slice for dependent_applications when input is nil, got nil")
	}

	if len(resp.DependentApplications) != 0 {
		t.Errorf("expected empty dependent_applications when input is nil, got %v", resp.DependentApplications)
	}
}

func TestToResponseWithInvalidDependentApplications(t *testing.T) {
	at := ApplicationType{
		Id:                    4,
		Name:                  "/insights/platform/invalid-test",
		DisplayName:           "Invalid Test",
		DependentApplications: datatypes.JSON(`not-valid-json`),
	}

	resp := at.ToResponse()

	// Invalid JSON should result in empty slice (graceful degradation)
	if resp.DependentApplications == nil {
		t.Error("expected non-nil empty slice for dependent_applications with invalid JSON, got nil")
	}

	if len(resp.DependentApplications) != 0 {
		t.Errorf("expected empty dependent_applications with invalid JSON, got %v", resp.DependentApplications)
	}
}

func TestGoodUrl(t *testing.T) {
	expected := "http://a.good/uri"
	os.Setenv("TEST_NAME_AVAILABILITY_CHECK_URL", expected)

	a := ApplicationType{Name: "/this/is/my/test-name"}
	uri := a.AvailabilityCheckURL()

	if uri.String() != expected {
		t.Errorf("got the wrong availability check url, got %v expected %v", uri.String(), expected)
	}
}

func TestNotExistingUrl(t *testing.T) {
	a := ApplicationType{Name: "/this/one/does/not/exist"}
	uri := a.AvailabilityCheckURL()

	if uri != nil {
		t.Errorf("uri pulled from ENV even though it does not exist")
	}
}
