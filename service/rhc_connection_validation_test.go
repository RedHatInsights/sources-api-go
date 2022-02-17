package service

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/model"
)

// TestRhcConnectionCreateEmptyRhcId tests that an error is returned when an empty RhcId is received from the request.
func TestRhcConnectionCreateEmptyRhcId(t *testing.T) {
	rhcConnectionCreateRequest := model.RhcConnectionCreateRequest{
		RhcId: "",
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

// TestCreateRequestInvalidSourceIdFormat tests if an error is returned when an invalid format or string is provided for
// the source id.
func TestCreateRequestInvalidSourceIdFormat(t *testing.T) {
	ecr := setUpEndpointCreateRequest()

	ecr.SourceIDRaw = "hello world"

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "the provided source ID is not valid"
	if err.Error() != want {
		t.Errorf("want '%s', got '%s'", want, err)
	}
}

// TestCreateRequestInvalidSourceId tests if an error is returned when providing a source id lower than one.
func TestCreateRequestInvalidSourceId(t *testing.T) {
	ecr := setUpEndpointCreateRequest()

	ecr.SourceIDRaw = "0"

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "invalid source id"
	if err.Error() != want {
		t.Errorf("want '%s', got '%s'", want, err)
	}
}
