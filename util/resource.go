package util

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/types"
)

var resourceTypeWithStringIDs = []string{"Authentication"}

type Resource struct {
	ResourceType  string
	ResourceID    int64
	ResourceUID   string
	TenantID      int64
	AccountNumber string
}

func ParseStatusMessageToResource(resource *Resource, statusMessage types.StatusMessage) (*Resource, error) {
	resource.ResourceUID = statusMessage.ResourceID
	resource.ResourceType = Capitalize(statusMessage.ResourceType)

	resourceID, err := InterfaceToInt64(statusMessage.ResourceID)
	isErr := err != nil
	resourceTypeWithStringID := SliceContainsString(resourceTypeWithStringIDs, statusMessage.ResourceType)
	if isErr && !resourceTypeWithStringID {
		return nil, fmt.Errorf("error in parsing resource: %v, error: %w", resource, err)
	}

	if resourceTypeWithStringID && statusMessage.ResourceID == "" {
		return nil, fmt.Errorf("statusMessage.ResourceID is empty: %v, error: %w", resource, err)
	}

	resource.ResourceID = resourceID
	return resource, nil
}

func FormatAvailabilityStatus(status string) string {
	if status == "" {
		return "unknown"
	}

	return status
}
