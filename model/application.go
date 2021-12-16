package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type Application struct {
	AvailabilityStatus
	Pause

	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AvailabilityStatusError string         `json:"availability_status_error,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`
	SuperkeyData            datatypes.JSON

	TenantID int64
	Tenant   Tenant

	SourceID int64 `json:"source_id"`
	Source   Source

	ApplicationTypeID int64 `json:"application_type_id"`
	ApplicationType   ApplicationType
}

func (app *Application) ToEvent() *ApplicationEvent {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(app.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.DateTimeToRecordFormat(app.LastAvailableAt),
		LastCheckedAt:   util.DateTimeToRecordFormat(app.LastCheckedAt)}

	appEvent := &ApplicationEvent{
		AvailabilityStatusEvent: asEvent,
		PauseEvent:              PauseEvent{PausedAt: util.DateTimeToRecordFormat(app.PausedAt)},
		Extra:                   app.Extra,
		ID:                      app.ID,
		CreatedAt:               util.DateTimeToRecordFormat(app.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(app.UpdatedAt),
		ApplicationTypeID:       app.ApplicationTypeID,
		AvailabilityStatusError: util.StringValueOrNil(app.AvailabilityStatusError),
		SourceID:                app.SourceID,
		Tenant:                  &app.Tenant.ExternalTenant,
	}

	return appEvent
}

func (app *Application) ToResponse() *ApplicationResponse {
	id := strconv.FormatInt(app.ID, 10)
	sourceId := strconv.FormatInt(app.SourceID, 10)
	appTypeId := strconv.FormatInt(app.ApplicationTypeID, 10)

	return &ApplicationResponse{
		AvailabilityStatus:      app.AvailabilityStatus,
		ID:                      id,
		CreatedAt:               app.CreatedAt,
		UpdatedAt:               app.UpdatedAt,
		Pause:                   app.Pause,
		AvailabilityStatusError: app.AvailabilityStatusError,
		Extra:                   app.Extra,
		SourceID:                sourceId,
		ApplicationTypeID:       appTypeId,
	}
}
