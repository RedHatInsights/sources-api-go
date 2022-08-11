package service

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestValidApplicationRequest(t *testing.T) {
	AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		SourceIDRaw:          "1",
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err != nil {
		t.Errorf("Got an error when there should not have been one: %v", err)
	}

	if req.ApplicationTypeID != 1 {
		t.Errorf("Failed to parse app type id, got %v, wanted 1", req.ApplicationTypeID)
	}

	if req.ApplicationTypeID != 1 {
		t.Errorf("Failed to parse app type id, got %v, wanted 1", req.ApplicationTypeID)
	}

	if req.SourceID != 1 {
		t.Errorf("Failed to parse source id, got %v, wanted 1", req.SourceID)
	}
}

func TestInvalidApplicationTypeForSource(t *testing.T) {
	AppTypeDao = &dao.MockApplicationTypeDao{Compatible: false}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		SourceIDRaw:          "1",
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("Got no error when there should not have been one")
	}

	if !strings.Contains(err.Error(), "is not compatible") {
		t.Errorf("malformed error message: %v", err)
	}
}

func TestMissingApplicationTypeId(t *testing.T) {
	AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:       []byte(`{"a good":"thing"`),
		SourceIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("No error when there should have been one")
	}
}

func TestMissingSourceId(t *testing.T) {
	AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("No error when there should have been one")
	}
}

// TestEditNilAvailabilityStatus tests that when a nil availability status is provided —simulating a JSON payload which
// doesn't contain that key—, no error is received from the validation function.
func TestEditNilAvailabilityStatus(t *testing.T) {
	editRequest := m.ApplicationEditRequest{
		AvailabilityStatus: nil,
	}

	err := ValidateApplicationEditRequest(&editRequest)

	if err != nil {
		t.Errorf(`unexpected error when validating an empty availability status for an application edit: %s`, err)
	}
}

// TestEditInvalidAvailabilityStatuses tests that a proper error is returned when providing invalid availability
// statuses.
func TestEditInvalidAvailabilityStatuses(t *testing.T) {
	testValues := []*string{
		util.StringRef(""),
		util.StringRef("availablel"),
		util.StringRef("inprogress"),
		util.StringRef("partial"),
		util.StringRef("unavalialbe"),
	}

	want := `availability status invalid. Must be one of "available", "in_progress", "partially_available" or "unavailable"`
	for _, tv := range testValues {
		editRequest := m.ApplicationEditRequest{
			AvailabilityStatus: tv,
		}

		err := ValidateApplicationEditRequest(&editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error when validating invalid availability statuses for an application edit. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestEditValidAvailabilityStatuses tests that no error is returned when valid availability statuses are provided.
func TestEditValidAvailabilityStatuses(t *testing.T) {
	testValues := []*string{
		util.StringRef(m.Available),
		util.StringRef(m.InProgress),
		util.StringRef(m.PartiallyAvailable),
		util.StringRef(m.Unavailable),
	}

	for _, tv := range testValues {
		editRequest := m.ApplicationEditRequest{
			AvailabilityStatus: tv,
		}

		err := ValidateApplicationEditRequest(&editRequest)

		if err != nil {
			t.Errorf(`unexpected error when validating a valid availability status "%s" for an application edit: %s`, *tv, err)
		}
	}
}

// TestEditInvalidAvailabilityStatusPaused tests that an error is received when an invalid availability status is given
// when updating a paused application.
func TestEditInvalidAvailabilityStatusPaused(t *testing.T) {
	testValues := []*string{
		util.StringRef(""),
		util.StringRef("availablel"),
		util.StringRef("inprogress"),
		util.StringRef("partial"),
		util.StringRef("unavalialbe"),
	}

	want := `invalid availability status. Must be one of "available", "in_progress", "partially_available" or "unavailable"`
	for _, tv := range testValues {

		editRequest := m.ResourceEditPausedRequest{
			AvailabilityStatus: tv,
		}

		app := m.Application{}
		err := app.UpdateFromRequestPaused(&editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error received when updating a paused application. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestEditValidAvailabilityStatusPaused tests that no error is returned when valid availability statuses are provided
// when updating a paused application.
func TestEditValidAvailabilityStatusPaused(t *testing.T) {
	testValues := []*string{
		util.StringRef(m.Available),
		util.StringRef(m.InProgress),
		util.StringRef(m.PartiallyAvailable),
		util.StringRef(m.Unavailable),
	}

	for _, tv := range testValues {
		editRequest := m.ResourceEditPausedRequest{
			AvailabilityStatus: tv,
		}

		app := m.Application{}
		err := app.UpdateFromRequestPaused(&editRequest)

		if err != nil {
			t.Errorf(`unexpected error when validating a valid availability status "%s" for a paused application edit: %s`, *tv, err)
		}
	}
}
