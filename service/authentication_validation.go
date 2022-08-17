package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

var validAuthenticationResources = []string{"source", "endpoint", "application", "authentication"}

func ValidateAuthenticationCreationRequest(auth *model.AuthenticationCreateRequest) error {
	if auth.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}
	if auth.ResourceIDRaw == nil {
		return fmt.Errorf("resource_id is required")
	}

	rid, err := util.InterfaceToInt64(auth.ResourceIDRaw)
	if err != nil {
		return fmt.Errorf("resource_id must be a valid integer or string")
	}
	auth.ResourceID = rid

	if !util.SliceContainsString(validAuthenticationResources, strings.ToLower(auth.ResourceType)) {
		return fmt.Errorf("invalid resource_type - must be one of [Source|Endpoint|Application|Authentication]")
	}

	// capitalize it so it's always the same format.
	auth.ResourceType = util.Capitalize(auth.ResourceType)

	return nil
}

func ValidateAuthenticationEditRequest(auth *model.AuthenticationEditRequest) error {
	if auth.AvailabilityStatus != nil {
		if _, ok := model.ValidAvailabilityStatuses[*auth.AvailabilityStatus]; !ok {
			return errors.New(`availability status invalid. Must be one of "available", "in_progress", "partially_available" or "unavailable"`)
		}
	}

	return nil
}
