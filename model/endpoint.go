package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
)

type Endpoint struct {
	ID        int64      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	PausedAt  *time.Time `json:"paused_at"`

	Role                 *string `json:"role,omitempty"`
	Port                 *int    `json:"port,omitempty"`
	Default              *bool   `json:"default,omitempty"`
	Scheme               *string `json:"scheme,omitempty"`
	Host                 *string `json:"host,omitempty"`
	Path                 *string `json:"path,omitempty"`
	VerifySsl            *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority *string `json:"certificate_authority,omitempty"`
	ReceptorNode         *string `json:"receptor_node,omitempty"`

	AvailabilityStatus      string     `json:"availability_status,omitempty"`
	LastCheckedAt           *time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt         *time.Time `json:"last_available_at,omitempty"`
	AvailabilityStatusError *string    `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`
	Source   Source

	TenantID int64
	Tenant   Tenant
}

func (endpoint *Endpoint) ToEvent() interface{} {
	endpointEvent := &EndpointEvent{
		PausedAt:                util.DateTimePointerToRecordFormat(endpoint.PausedAt),
		ID:                      endpoint.ID,
		CertificateAuthority:    endpoint.CertificateAuthority,
		Host:                    endpoint.Host,
		Port:                    endpoint.Port,
		ReceptorNode:            endpoint.ReceptorNode,
		Role:                    endpoint.Role,
		Scheme:                  endpoint.Scheme,
		SourceID:                endpoint.SourceID,
		VerifySsl:               endpoint.VerifySsl,
		Default:                 endpoint.Default,
		Path:                    endpoint.Path,
		CreatedAt:               util.DateTimeToRecordFormat(endpoint.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(endpoint.UpdatedAt),
		AvailabilityStatus:      util.StringValueOrNil(endpoint.AvailabilityStatus),
		LastAvailableAt:         util.DateTimePointerToRecordFormat(endpoint.LastAvailableAt),
		LastCheckedAt:           util.DateTimePointerToRecordFormat(endpoint.LastCheckedAt),
		AvailabilityStatusError: util.StringValueOrNil(endpoint.AvailabilityStatusError),
		Tenant:                  &endpoint.Tenant.ExternalTenant,
	}

	return endpointEvent
}

func (endpoint *Endpoint) ToResponse() *EndpointResponse {
	id := strconv.FormatInt(endpoint.ID, 10)
	sourceId := strconv.FormatInt(endpoint.SourceID, 10)

	return &EndpointResponse{
		ID:                      id,
		CreatedAt:               util.DateTimeToRFC3339(endpoint.CreatedAt),
		UpdatedAt:               util.DateTimeToRFC3339(endpoint.UpdatedAt),
		PausedAt:                util.DateTimePointerToRFC3339(endpoint.PausedAt),
		Role:                    endpoint.Role,
		Port:                    endpoint.Port,
		Default:                 endpoint.Default,
		Scheme:                  endpoint.Scheme,
		Host:                    endpoint.Host,
		Path:                    endpoint.Path,
		VerifySsl:               endpoint.VerifySsl,
		CertificateAuthority:    endpoint.CertificateAuthority,
		ReceptorNode:            endpoint.ReceptorNode,
		AvailabilityStatus:      util.StringValueOrNil(endpoint.AvailabilityStatus),
		LastCheckedAt:           util.DateTimePointerToRFC3339(endpoint.LastCheckedAt),
		LastAvailableAt:         util.DateTimePointerToRFC3339(endpoint.LastAvailableAt),
		AvailabilityStatusError: endpoint.AvailabilityStatusError,
		SourceID:                sourceId,
	}
}
