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
	AvailabilityStatus string     `json:"availability_status,omitempty"`
	LastCheckedAt      *time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt    *time.Time `json:"last_available_at,omitempty"`

	//fields for gorm
	ID        int64      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	PausedAt  *time.Time `json:"paused_at"`

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
	User     User
	UserID   *int64 `json:"user_id"`

	ApplicationTypes     []*ApplicationType `gorm:"many2many:applications"`
	Applications         []Application
	Endpoints            []Endpoint
	Authentications      []Authentication `gorm:"-"`
	SourceRhcConnections []SourceRhcConnection
}

func (src *Source) ToEvent() interface{} {
	sourceEvent := &SourceEvent{
		ID:                  &src.ID,
		CreatedAt:           util.DateTimeToRecordFormat(src.CreatedAt),
		UpdatedAt:           util.DateTimeToRecordFormat(src.UpdatedAt),
		PausedAt:            util.DateTimePointerToRecordFormat(src.PausedAt),
		AvailabilityStatus:  util.StringValueOrNil(src.AvailabilityStatus),
		LastAvailableAt:     util.DateTimePointerToRecordFormat(src.LastAvailableAt),
		LastCheckedAt:       util.DateTimePointerToRecordFormat(src.LastCheckedAt),
		Name:                &src.Name,
		UID:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: &src.AppCreationWorkflow,
		SourceTypeID:        &src.SourceTypeID,
		Tenant:              &src.Tenant.ExternalTenant,
	}

	return sourceEvent
}

func (src *Source) ToResponse() *SourceResponse {
	id := strconv.FormatInt(src.ID, 10)
	stid := strconv.FormatInt(src.SourceTypeID, 10)

	return &SourceResponse{
		AvailabilityStatus:  util.StringValueOrNil(src.AvailabilityStatus),
		LastCheckedAt:       util.DateTimePointerToRFC3339(src.LastCheckedAt),
		LastAvailableAt:     util.DateTimePointerToRFC3339(src.LastAvailableAt),
		ID:                  id,
		CreatedAt:           util.DateTimeToRFC3339(src.CreatedAt),
		UpdatedAt:           util.DateTimeToRFC3339(src.UpdatedAt),
		PausedAt:            util.DateTimePointerToRFC3339(src.PausedAt),
		Name:                &src.Name,
		Uid:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: &src.AppCreationWorkflow,
		SourceTypeId:        stid,
	}
}

// ToInternalResponse returns only the fields that "sources-monitor-go" requires.
func (src *Source) ToInternalResponse() *SourceInternalResponse {
	id := strconv.FormatInt(src.ID, 10)

	source := &SourceInternalResponse{
		Id:                 &id,
		AvailabilityStatus: &src.AvailabilityStatus,
		ExternalTenant:     &src.Tenant.ExternalTenant,
	}

	return source
}

func (src *Source) IsSuperkey() bool {
	return src.AppCreationWorkflow == AccountAuth
}

func (src *Source) ToEmail(previousStatus string) *EmailNotificationInfo {
	return &EmailNotificationInfo{
		SourceName:                 src.Name,
		SourceID:                   strconv.FormatInt(src.ID, 10),
		ResourceDisplayName:        "Source",
		CurrentAvailabilityStatus:  util.FormatAvailabilityStatus(src.AvailabilityStatus),
		PreviousAvailabilityStatus: util.FormatAvailabilityStatus(previousStatus),
		TenantID:                   strconv.FormatInt(src.TenantID, 10),
	}
}
