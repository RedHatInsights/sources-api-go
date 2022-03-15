package model

import (
	"gorm.io/datatypes"
)

type ApplicationResponse struct {
	AvailabilityStatusResponse
	PauseResponse

	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

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
}
