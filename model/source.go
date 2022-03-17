package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
)

// App creation workflow's constants
const (
	AccountAuth  string = "account_authorization"
	ManualConfig string = "manual_configuration"
)

// Source struct that includes all of the fields on the table
// used internally for business logic
type Source struct {
	AvailabilityStatus

	//fields for gorm
	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PausedAt  time.Time `json:"paused_at"`

	// standard source fields
	Name                string  `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow string  `gorm:"default:manual_configuration" json:"app_creation_workflow"`

	SourceType   SourceType
	SourceTypeID int64 `json:"source_type_id"`

	Tenant   Tenant
	TenantID int64 `json:"tenant_id"`

	ApplicationTypes []*ApplicationType `gorm:"many2many:applications"`
	Applications     []Application
	Endpoints        []Endpoint
	Authentications  []Authentication `gorm:"-"`
}

func (src *Source) ToEvent() interface{} {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(src.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.DateTimeToRecordFormat(src.LastAvailableAt),
		LastCheckedAt:   util.DateTimeToRecordFormat(src.LastCheckedAt)}

	sourceEvent := &SourceEvent{
		AvailabilityStatusEvent: asEvent,
		PausedAt:                util.DateTimeToRecordFormat(src.PausedAt),
		ID:                      &src.ID,
		CreatedAt:               util.DateTimeToRecordFormat(src.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(src.UpdatedAt),
		Name:                    &src.Name,
		UID:                     src.Uid,
		Version:                 src.Version,
		Imported:                src.Imported,
		SourceRef:               src.SourceRef,
		AppCreationWorkflow:     &src.AppCreationWorkflow,
		SourceTypeID:            &src.SourceTypeID,
		Tenant:                  &src.Tenant.ExternalTenant,
	}

	return sourceEvent
}

func (src *Source) ToResponse() *SourceResponse {
	id := strconv.FormatInt(src.ID, 10)
	stid := strconv.FormatInt(src.SourceTypeID, 10)
	asResponse := AvailabilityStatusResponse{
		AvailabilityStatus: util.StringValueOrNil(src.AvailabilityStatus.AvailabilityStatus),
		LastCheckedAt:      util.DateTimeToRFC3339(src.LastCheckedAt),
		LastAvailableAt:    util.DateTimeToRFC3339(src.LastAvailableAt),
	}

	return &SourceResponse{
		AvailabilityStatusResponse: asResponse,
		ID:                         id,
		CreatedAt:                  util.DateTimeToRFC3339(src.CreatedAt),
		UpdatedAt:                  util.DateTimeToRFC3339(src.UpdatedAt),
		PausedAt:                   util.DateTimeToRFC3339(src.PausedAt),
		Name:                       &src.Name,
		Uid:                        src.Uid,
		Version:                    src.Version,
		Imported:                   src.Imported,
		SourceRef:                  src.SourceRef,
		AppCreationWorkflow:        &src.AppCreationWorkflow,
		SourceTypeId:               stid,
	}
}

// ToInternalResponse returns only the fields that "sources-monitor-go" requires.
func (src *Source) ToInternalResponse() *SourceInternalResponse {
	id := strconv.FormatInt(src.ID, 10)

	source := &SourceInternalResponse{
		Id:                 &id,
		AvailabilityStatus: &src.AvailabilityStatus.AvailabilityStatus,
		ExternalTenant:     &src.Tenant.ExternalTenant,
	}

	return source
}

func (src *Source) IsSuperkey() bool {
	return src.AppCreationWorkflow == AccountAuth
}
