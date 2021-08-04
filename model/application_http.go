package model

import (
	"time"

	"gorm.io/datatypes"
)

type ApplicationResponse struct {
	AvailabilityStatus
	Pause

	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AvailabilityStatusError string         `json:"availability_status_error,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`

	SourceID          string `json:"source_id"`
	ApplicationTypeID string `json:"application_type_id"`
}
