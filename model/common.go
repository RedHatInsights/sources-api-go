package model

import "time"

type AvailabilityStatus struct {
	AvailabilityStatus string    `json:"availability_status,omitempty"`
	LastCheckedAt      time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt    time.Time `json:"last_available_at,omitempty"`
}

type AvailabilityStatusResponse struct {
	AvailabilityStatus *string `json:"availability_status,omitempty"`
	LastCheckedAt      string  `json:"last_checked_at,omitempty"`
	LastAvailableAt    string  `json:"last_available_at,omitempty"`
}
