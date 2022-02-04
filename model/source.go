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
	Pause

	//fields for gorm
	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

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
}

func (src *Source) ToEvent() interface{} {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(src.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.DateTimeToRecordFormat(src.LastAvailableAt),
		LastCheckedAt:   util.DateTimeToRecordFormat(src.LastCheckedAt)}

	sourceEvent := &SourceEvent{
		AvailabilityStatusEvent: asEvent,
		PauseEvent:              PauseEvent{PausedAt: util.DateTimeToRecordFormat(src.PausedAt)},
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

	return &SourceResponse{
		AvailabilityStatus:  src.AvailabilityStatus,
		Pause:               src.Pause,
		ID:                  &id,
		CreatedAt:           src.CreatedAt,
		UpdatedAt:           src.UpdatedAt,
		Name:                &src.Name,
		Uid:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: &src.AppCreationWorkflow,
		SourceTypeId:        &stid,
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
