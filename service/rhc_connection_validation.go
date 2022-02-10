package service

import (
	"errors"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// ValidateRhcConnectionRequest validates that the incoming input is valid.
func ValidateRhcConnectionRequest(req *model.RhcConnectionCreateRequest) error {
	if !util.SliceContainsString(model.AvailabilityStatuses, req.AvailabilityStatus) {
		return errors.New("invalid availability status")
	}

	if req.RhcId == "" {
		return errors.New("the Red Hat Connector Connection's id is invalid")
	}

	return nil
}
