package service

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// ValidateRhcConnectionRequest validates that the incoming input is valid.
func ValidateRhcConnectionRequest(req *model.RhcConnectionCreateRequest) error {
	if req.RhcId == "" {
		return errors.New("the Red Hat Connector Connection's id is invalid")
	}

	sourceId, err := util.InterfaceToInt64(req.SourceId)
	if err != nil {
		return fmt.Errorf("the provided source ID is not valid")
	}

	if sourceId < 1 {
		return fmt.Errorf("invalid source id")
	}

	req.SourceId = sourceId

	return nil
}
