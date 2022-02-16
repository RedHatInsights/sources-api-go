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
