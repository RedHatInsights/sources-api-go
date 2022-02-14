package model

import (
	"gorm.io/datatypes"
)

// RhcConnectionCreateRequest represents a request coming from the outside to create a Red Hat Connector connection.
type RhcConnectionCreateRequest struct {
	RhcId              string         `json:"rhc_id"`
	Extra              datatypes.JSON `json:"extra"`
	AvailabilityStatus string         `json:"availability_status"`
	SourceId           string         `json:"source_id"`
}

// RhcConnectionUpdateRequest represents a request coming from the outside to update a Red Hat Connector connection.
type RhcConnectionUpdateRequest struct {
	Extra datatypes.JSON `json:"extra"`
}

// RhcConnectionListResponse doesn't include the source ID, since it would be very expensive to get all the sources for
// all the RhcConnections that a user might want to list.
type RhcConnectionListResponse struct {
	Uuid  *string        `json:"rhc_id"`
	Extra datatypes.JSON `json:"extra,omitempty"`
	AvailabilityStatus
	AvailabilityStatusError string `json:"availability_status_error,omitempty"`
}

type RhcConnectionResponse struct {
	Uuid  *string        `json:"rhc_id"`
	Extra datatypes.JSON `json:"extra,omitempty"`
	AvailabilityStatus
	AvailabilityStatusError string `json:"availability_status_error,omitempty"`
	SourceId                string `json:"source_id"`
}
