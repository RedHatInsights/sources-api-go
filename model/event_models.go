package model

import (
	"time"

	"gorm.io/datatypes"
)

type PauseEvent struct {
	PausedAt *string `json:"paused_at"`
}

type AvailabilityStatusEvent struct {
	AvailabilityStatus *string `json:"availability_status"`
	LastCheckedAt      *string `json:"last_checked_at"`
	LastAvailableAt    *string `json:"last_available_at"`
}

type ApplicationEvent struct {
	AvailabilityStatusEvent
	PauseEvent

	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`

	AvailabilityStatusError *string        `json:"availability_status_error"`
	Extra                   datatypes.JSON `json:"extra"`
	SuperkeyData            datatypes.JSON `json:"superkey_data"`

	SourceID          int64   `json:"source_id"`
	ApplicationTypeID int64   `json:"application_type_id"`
	Tenant            *string `json:"tenant"`
}

type AuthenticationEvent struct {
	AvailabilityStatusEvent

	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name                    string                 `json:"name"`
	AuthType                string                 `json:"authtype"`
	Username                string                 `json:"username"`
	Extra                   map[string]interface{} `json:"extra"`
	Version                 string                 `json:"version"`
	AvailabilityStatusError *string                `json:"availability_status_error"`
	ResourceType            string                 `json:"resource_type"`
	ResourceID              int64                  `json:"resource_id"`
	SourceID                int64                  `json:"source_id"`
	Tenant                  *string                `json:"tenant"`
}

type ApplicationAuthenticationEvent struct {
	PauseEvent

	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`

	ApplicationID     int64  `json:"application_id"`
	AuthenticationID  int64  `json:"authentication_id"`
	AuthenticationUID string `json:"authentication_uid"`

	Tenant    *string `json:"tenant"`
	VaultPath string  `json:"vault_path"`
}

type EndpointEvent struct {
	AvailabilityStatusEvent
	PauseEvent

	ID        int64   `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`

	Role                    *string `json:"role"`
	Port                    *int    `json:"port"`
	Default                 *bool   `json:"default"`
	Scheme                  *string `json:"scheme"`
	Host                    *string `json:"host"`
	Path                    *string `json:"path"`
	VerifySsl               *bool   `json:"verify_ssl"`
	CertificateAuthority    *string `json:"certificate_authority"`
	ReceptorNode            *string `json:"receptor_node"`
	AvailabilityStatusError *string `json:"availability_status_error"`

	SourceID int64   `json:"source_id"`
	Tenant   *string `json:"tenant"`
}

type SourceEvent struct {
	AvailabilityStatusEvent
	PauseEvent

	ID                  *int64  `json:"id"`
	CreatedAt           *string `json:"created_at"`
	UpdatedAt           *string `json:"updated_at"`
	Name                *string `json:"name"`
	UID                 *string `json:"uid"`
	Version             *string `json:"version"`
	Imported            *string `json:"imported"`
	SourceRef           *string `json:"source_ref"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`
	SourceTypeID        *int64  `json:"source_type_id"`
	Tenant              *string `json:"tenant"`
}
