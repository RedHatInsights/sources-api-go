package model

import (
	"time"

	"gorm.io/datatypes"
)

type ApplicationEvent struct {
	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
	PausedAt  *string `json:"paused_at"`

	AvailabilityStatus      *string        `json:"availability_status"`
	LastCheckedAt           *string        `json:"last_checked_at"`
	LastAvailableAt         *string        `json:"last_available_at"`
	AvailabilityStatusError *string        `json:"availability_status_error"`
	Extra                   datatypes.JSON `json:"extra"`
	SuperkeyData            datatypes.JSON `json:"superkey_data"`

	SourceID          int64   `json:"source_id"`
	ApplicationTypeID int64   `json:"application_type_id"`
	Tenant            *string `json:"tenant"`
}

type AuthenticationEvent struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name                    string                 `json:"name"`
	AuthType                string                 `json:"authtype"`
	Username                string                 `json:"username"`
	Extra                   map[string]interface{} `json:"extra"`
	Version                 string                 `json:"version"`
	AvailabilityStatus      *string                `json:"availability_status"`
	LastCheckedAt           *string                `json:"last_checked_at"`
	LastAvailableAt         *string                `json:"last_available_at"`
	AvailabilityStatusError *string                `json:"availability_status_error"`
	ResourceType            string                 `json:"resource_type"`
	ResourceID              int64                  `json:"resource_id"`
	SourceID                int64                  `json:"source_id"`
	Tenant                  *string                `json:"tenant"`
}

type ApplicationAuthenticationEvent struct {
	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
	PausedAt  *string `json:"paused_at"`

	ApplicationID    int64 `json:"application_id"`
	AuthenticationID int64 `json:"authentication_id"`
	// TODO: maybe add back in if we get vault
	AuthenticationUID string `json:"-"`

	Tenant *string `json:"tenant"`
	// TODO: add back in if we get vault
	VaultPath string `json:"-"`
}

type EndpointEvent struct {
	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
	PausedAt  *string `json:"paused_at"`

	Role                    *string `json:"role"`
	Port                    *int    `json:"port"`
	Default                 *bool   `json:"default"`
	Scheme                  *string `json:"scheme"`
	Host                    *string `json:"host"`
	Path                    *string `json:"path"`
	VerifySsl               *bool   `json:"verify_ssl"`
	CertificateAuthority    *string `json:"certificate_authority"`
	ReceptorNode            *string `json:"receptor_node"`
	AvailabilityStatus      *string `json:"availability_status"`
	LastCheckedAt           *string `json:"last_checked_at"`
	LastAvailableAt         *string `json:"last_available_at"`
	AvailabilityStatusError *string `json:"availability_status_error"`

	SourceID int64   `json:"source_id"`
	Tenant   *string `json:"tenant"`
}

type SourceEvent struct {
	AvailabilityStatus *string `json:"availability_status"`
	LastCheckedAt      *string `json:"last_checked_at"`
	LastAvailableAt    *string `json:"last_available_at"`

	ID                  *int64  `json:"id"`
	CreatedAt           *string `json:"created_at"`
	UpdatedAt           *string `json:"updated_at"`
	PausedAt            *string `json:"paused_at"`
	Name                *string `json:"name"`
	UID                 *string `json:"uid"`
	Version             *string `json:"version"`
	Imported            *string `json:"imported"`
	SourceRef           *string `json:"source_ref"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`
	SourceTypeID        *int64  `json:"source_type_id"`
	Tenant              *string `json:"tenant"`
}

type RhcConnectionEvent struct {
	ID                      *int64         `json:"id"`
	RhcId                   *string        `json:"rhc_id"`
	Extra                   datatypes.JSON `json:"extra"`
	AvailabilityStatus      *string        `json:"availability_status"`
	LastCheckedAt           *string        `json:"last_checked_at"`
	LastAvailableAt         *string        `json:"last_available_at"`
	AvailabilityStatusError *string        `json:"availability_status_error"`
	SourceIds               []string       `json:"source_ids"`

	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}
