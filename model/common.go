package model

import "time"

type TimeStamps struct {
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type Pause struct {
	PausedAt time.Time `json:"paused_at,omitempty"`
}

type AvailabilityStatus struct {
	AvailabilityStatus string    `json:"availability_status,omitempty"`
	LastCheckedAt      time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt    time.Time `json:"last_available_at,omitempty"`
}
