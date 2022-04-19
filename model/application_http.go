package model

import (
	"gorm.io/datatypes"
)

type ApplicationResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	PausedAt  string `json:"paused_at,omitempty"`

	AvailabilityStatus      *string        `json:"availability_status,omitempty"`
	LastCheckedAt           string         `json:"last_checked_at,omitempty"`
	LastAvailableAt         string         `json:"last_available_at,omitempty"`
	AvailabilityStatusError string         `json:"availability_status_error,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`

	SourceID          string `json:"source_id"`
	ApplicationTypeID string `json:"application_type_id"`
}

type ApplicationCreateRequest struct {
	Extra datatypes.JSON `json:"extra,omitempty"`

	SourceID             int64       `json:"-"`
	SourceIDRaw          interface{} `json:"source_id"`
	ApplicationTypeID    int64       `json:"-"`
	ApplicationTypeIDRaw interface{} `json:"application_type_id"`
}

type ApplicationEditRequest struct {
	Extra map[string]interface{} `json:"extra,omitempty"`

	AvailabilityStatus      *string `json:"availability_status"`
	AvailabilityStatusError *string `json:"availability_status_error"`

	// TODO: remove these once satellite goes away.
	LastCheckedAt   *string `json:"last_checked_at"`
	LastAvailableAt *string `json:"last_available_at"`
}
