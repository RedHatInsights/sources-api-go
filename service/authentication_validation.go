package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
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

	err = ValidateAzureSubscriptionId(auth)
	if err != nil {
		return fmt.Errorf("subscription ID is invalid: %w", err)
	}

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

func ValidateAzureSubscriptionId(auth *model.AuthenticationCreateRequest) error {
	if auth == nil {
		return errors.New("auth is nil")
	}

	if auth.AuthType == "provisioning_lighthouse_subscription_id" || auth.AuthType == "lighthouse_subscription_id" {
		if auth.Username == nil {
			return errors.New("username is required for Azure Source Types")
		}

		trimmed := strings.TrimSpace(*auth.Username)
		if trimmed == "" {
			return errors.New("username must not be blank or empty for Azure Source Types")
		}

		_, err := uuid.Parse(trimmed)
		if err != nil {
			return fmt.Errorf("the username must be a valid UUID for Azure Source Types")
		}
	}

	return nil
}
