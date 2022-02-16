package service

import (
	"errors"

	"github.com/RedHatInsights/sources-api-go/model"
)

// ValidateRhcConnectionRequest validates that the incoming input is valid.
func ValidateRhcConnectionRequest(req *model.RhcConnectionCreateRequest) error {
	if req.RhcId == "" {
		return errors.New("the Red Hat Connector Connection's id is invalid")
	}

	return nil
}
