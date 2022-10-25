package model

import (
	"time"
)

type AuthenticationResponse struct {
	ID string `json:"id"`

	Name                    string                 `json:"name,omitempty"`
	AuthType                string                 `json:"authtype"`
	Username                string                 `json:"username"`
	Extra                   map[string]interface{} `json:"extra,omitempty"`
	AvailabilityStatus      string                 `json:"availability_status,omitempty"`
	AvailabilityStatusError string                 `json:"availability_status_error,omitempty"`

	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type AuthenticationInternalResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name                    string                 `json:"name,omitempty"`
	AuthType                string                 `json:"authtype"`
	Username                string                 `json:"username"`
	Password                string                 `json:"password,omitempty"`
	Extra                   map[string]interface{} `json:"extra,omitempty"`
	Version                 string                 `json:"version"`
	AvailabilityStatus      string                 `json:"availability_status,omitempty"`
	AvailabilityStatusError string                 `json:"availability_status_error,omitempty"`

	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type AuthenticationCreateRequest struct {
	Name                    *string                `json:"name,omitempty"`
	AuthType                string                 `json:"authtype"`
	Username                *string                `json:"username"`
	Password                *string                `json:"password,omitempty"`
	Extra                   map[string]interface{} `json:"extra,omitempty"`
	AvailabilityStatusError *string                `json:"availability_status_error,omitempty"`

	ResourceType  string      `json:"resource_type"`
	ResourceIDRaw interface{} `json:"resource_id"`
	ResourceID    int64       `json:"-"`
}

type AuthenticationEditRequest struct {
	Name                    *string                 `json:"name"`
	AuthType                *string                 `json:"authtype"`
	Username                *string                 `json:"username"`
	Password                *string                 `json:"password,omitempty"`
	Extra                   *map[string]interface{} `json:"extra,omitempty"`
	AvailabilityStatus      *string                 `json:"availability_status,omitempty"`
	AvailabilityStatusError *string                 `json:"availability_status_error,omitempty"`
}

func (auth *Authentication) UpdateFromRequest(update *AuthenticationEditRequest) error {
	if update.Name != nil {
		auth.Name = update.Name
	}
	if update.AuthType != nil {
		auth.AuthType = *update.AuthType
	}
	if update.Username != nil {
		auth.Username = update.Username
	}
	if update.Password != nil {
		err := auth.SetPassword(*update.Password)
		if err != nil {
			return err
		}
	}

	if update.Extra != nil {
		err := auth.SetExtra(*update.Extra)
		if err != nil {
			return err
		}
	}

	if update.AvailabilityStatus != nil {
		auth.AvailabilityStatus = update.AvailabilityStatus
	}
	if update.AvailabilityStatusError != nil {
		auth.AvailabilityStatusError = update.AvailabilityStatusError
	}

	return nil
}

func (auth *Authentication) UpdateSecretFromRequest(update *SecretEditRequest) error {
	if update.Username != nil {
		auth.Username = update.Username
	}

	if update.Password != nil {
		err := auth.SetPassword(*update.Password)
		if err != nil {
			return err
		}
	}

	if update.Extra != nil {
		err := auth.SetExtra(*update.Extra)
		if err != nil {
			return err
		}
	}

	return nil
}
