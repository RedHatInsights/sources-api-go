package service

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/model"
)

// TestRhcConnectionCreateValidAvailabilityStatuses tests if no error is returned when valid availability statuses are given.
func TestRhcConnectionValidAvailabilityStatuses(t *testing.T) {
	rhcConnectionCreateRequest := model.RhcConnectionCreateRequest{
		RhcId: "valid",
	}

	testValues := []string{"", model.Available, model.Unavailable}
	for _, tt := range testValues {
		rhcConnectionCreateRequest.AvailabilityStatus = tt

		err := ValidateRhcConnectionRequest(&rhcConnectionCreateRequest)
		if err != nil {
			t.Errorf(`want no error, got "%s"`, err)
		}
	}
}

// TestRhcConnectionCreateInvalidAvailabilityStatuses tests if an error is returned when passing invalid availability
// statuses.
func TestRhcConnectionCreateInvalidAvailabilityStatuses(t *testing.T) {
	rhcConnectionCreateRequest := model.RhcConnectionCreateRequest{
		RhcId: "valid",
	}

	invalidValues := []string{"hello", "world", "almost", "passes", "validation"}
	want := "invalid availability status"
	for _, tt := range invalidValues {
		rhcConnectionCreateRequest.AvailabilityStatus = tt

		err := ValidateRhcConnectionRequest(&rhcConnectionCreateRequest)
		if err.Error() != want {
			t.Errorf(`want "%s", got "%s"`, want, err)
		}
	}
}

func TestRhcConnectionCreateEmptyRhcId(t *testing.T) {
	rhcConnectionCreateRequest := model.RhcConnectionCreateRequest{
		AvailabilityStatus: "available",
		RhcId:              "",
	}

	err := ValidateRhcConnectionRequest(&rhcConnectionCreateRequest)
	if err == nil {
		t.Errorf(`want error, got nil`)
	}

	want := "the Red Hat Connector Connection's id is invalid"
	if want != err.Error() {
		t.Errorf(`want "%s", got "%s"`, want, err)
	}
}
