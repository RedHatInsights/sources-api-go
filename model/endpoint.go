package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
)

type Endpoint struct {
	AvailabilityStatus
	Pause

	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Role                    *string `json:"role,omitempty"`
	Port                    *int    `json:"port,omitempty"`
	Default                 *bool   `json:"default,omitempty"`
	Scheme                  *string `json:"scheme,omitempty"`
	Host                    *string `json:"host,omitempty"`
	Path                    *string `json:"path,omitempty"`
	VerifySsl               *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority    *string `json:"certificate_authority,omitempty"`
	ReceptorNode            *string `json:"receptor_node,omitempty"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`
	Source   Source

	TenantID int64
	Tenant   Tenant
}

func (endpoint *Endpoint) ToEvent() *EndpointEvent {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(endpoint.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.StringValueOrNil(util.FormatTimeToString(endpoint.LastAvailableAt, util.RecordDateTimeFormat)),
		LastCheckedAt:   util.StringValueOrNil(util.FormatTimeToString(endpoint.LastCheckedAt, util.RecordDateTimeFormat))}

	endpointEvent := &EndpointEvent{
		AvailabilityStatusEvent: asEvent,
		PauseEvent:              PauseEvent{PausedAt: util.StringValueOrNil(util.FormatTimeToString(endpoint.PausedAt, util.RecordDateTimeFormat))},
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
		CreatedAt:               util.StringValueOrNil(util.FormatTimeToString(endpoint.CreatedAt, util.RecordDateTimeFormat)),
		UpdatedAt:               util.StringValueOrNil(util.FormatTimeToString(endpoint.UpdatedAt, util.RecordDateTimeFormat)),
		AvailabilityStatusError: util.StringValueOrNil(endpoint.AvailabilityStatusError),
		Tenant:                  &endpoint.Tenant.ExternalTenant,
	}

	return endpointEvent
}

func (endpoint *Endpoint) ToResponse() *EndpointResponse {
	id := strconv.FormatInt(endpoint.ID, 10)
	sourceId := strconv.FormatInt(endpoint.SourceID, 10)

	return &EndpointResponse{
		AvailabilityStatus:      endpoint.AvailabilityStatus,
		ID:                      id,
		CreatedAt:               endpoint.CreatedAt,
		UpdatedAt:               endpoint.UpdatedAt,
		Pause:                   endpoint.Pause,
		Role:                    endpoint.Role,
		Port:                    endpoint.Port,
		Default:                 endpoint.Default,
		Scheme:                  endpoint.Scheme,
		Host:                    endpoint.Host,
		Path:                    endpoint.Path,
		VerifySsl:               endpoint.VerifySsl,
		CertificateAuthority:    endpoint.CertificateAuthority,
		ReceptorNode:            endpoint.ReceptorNode,
		AvailabilityStatusError: endpoint.AvailabilityStatusError,
		SourceID:                sourceId,
	}
}
