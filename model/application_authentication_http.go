package model

import (
	"time"
)

type ApplicationAuthenticationResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ApplicationID    string `json:"application_id"`
	AuthenticationID string `json:"authentication_id"`
}
