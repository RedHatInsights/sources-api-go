package service

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestIntId(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: 17,
		ResourceType:  "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}
}

func TestStringId(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidResourceType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "Thing",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestCapitalizeResource(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}

	if acr.ResourceType != "Source" {
		t.Errorf("Resource Type not sanitized")
	}
}

func TestMissingResourceType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestMissingResourceID(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceType: "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestInvalidIdType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceType:  "Source",
		ResourceIDRaw: struct{}{},
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// TestAuthEditInvalidAvailabilityStatuses tests that a proper error is returned when providing invalid availability
// statuses.
func TestAuthEditInvalidAvailabilityStatuses(t *testing.T) {
	testValues := []*string{
		util.StringRef(""),
		util.StringRef("availablel"),
		util.StringRef("inprogress"),
		util.StringRef("partial"),
		util.StringRef("unavalialbe"),
	}

	want := `availability status invalid. Must be one of "available", "in_progress", "partially_available" or "unavailable"`
	for _, tv := range testValues {
		editRequest := model.AuthenticationEditRequest{
			AvailabilityStatus: tv,
		}

		err := ValidateAuthenticationEditRequest(&editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error when validating invalid availability statuses for an application edit. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestAuthEditValidAvailabilityStatuses tests that no error is returned when valid availability statuses are provided.
func TestAuthEditValidAvailabilityStatuses(t *testing.T) {
	testValues := []*string{
		util.StringRef(model.Available),
		util.StringRef(model.InProgress),
		util.StringRef(model.PartiallyAvailable),
		util.StringRef(model.Unavailable),
	}

	for _, tv := range testValues {
		editRequest := model.AuthenticationEditRequest{
			AvailabilityStatus: tv,
		}

		err := ValidateAuthenticationEditRequest(&editRequest)

		if err != nil {
			t.Errorf(`unexpected error when validating a valid availability status "%s" for an application edit: %s`, *tv, err)
		}
	}
}
