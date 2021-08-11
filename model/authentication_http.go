package model

import (
	"time"

	"gorm.io/datatypes"
)

type AuthenticationResponse struct {
	//AvailabilityStatus
	//Pause

	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	Name                    string         `json:"name,omitempty"`
	AuthType                string         `json:"authtype"`
	Username                string         `json:"username"`
	Password                string         `json:"password,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`
	Version                 string         `json:"version"`
	AvailabilityStatusError string         `json:"availability_status_error,omitempty"`

	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type AuthenticationCreateRequest struct {
	Name                    string                 `json:"name,omitempty"`
	AuthType                string                 `json:"authtype"`
	Username                string                 `json:"username"`
	Password                string                 `json:"password,omitempty"`
	Extra                   map[string]interface{} `json:"extra,omitempty"`
	AvailabilityStatusError string                 `json:"availability_status_error,omitempty"`

	ResourceType string `json:"resource_type"`
	ResourceID   int64  `json:"resource_id"`
}
