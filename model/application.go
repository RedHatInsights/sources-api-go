package model

import (
	"strconv"
	"time"

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
