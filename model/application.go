package model

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type Application struct {
	ID        int64      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	PausedAt  *time.Time `json:"paused_at"`

	AvailabilityStatus      string     `json:"availability_status,omitempty"`
	LastCheckedAt           *time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt         *time.Time `json:"last_available_at,omitempty"`
	AvailabilityStatusError string     `json:"availability_status_error,omitempty"`

	Extra             datatypes.JSON `json:"extra,omitempty"`
	SuperkeyData      datatypes.JSON `json:"-"`
	GotSuperkeyUpdate bool           `json:"-" gorm:"-"`

	TenantID int64
	Tenant   Tenant

	SourceID int64 `json:"source_id"`
	Source   Source

	ApplicationTypeID int64 `json:"application_type_id"`
	ApplicationType   ApplicationType

	ApplicationAuthentications []ApplicationAuthentication
}

func (app *Application) ToEvent() interface{} {
	appEvent := &ApplicationEvent{
		ID:                      app.ID,
		Extra:                   app.Extra,
		CreatedAt:               util.DateTimeToRecordFormat(app.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(app.UpdatedAt),
		PausedAt:                util.DateTimePointerToRecordFormat(app.PausedAt),
		ApplicationTypeID:       app.ApplicationTypeID,
		AvailabilityStatus:      util.StringValueOrNil(app.AvailabilityStatus),
		LastAvailableAt:         util.DateTimePointerToRecordFormat(app.LastAvailableAt),
		LastCheckedAt:           util.DateTimePointerToRecordFormat(app.LastCheckedAt),
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
		ID:                      id,
		CreatedAt:               util.DateTimeToRFC3339(app.CreatedAt),
		UpdatedAt:               util.DateTimeToRFC3339(app.UpdatedAt),
		PausedAt:                util.DateTimePointerToRFC3339(app.PausedAt),
		AvailabilityStatus:      util.StringValueOrNil(app.AvailabilityStatus),
		LastCheckedAt:           util.DateTimePointerToRFC3339(app.LastCheckedAt),
		LastAvailableAt:         util.DateTimePointerToRFC3339(app.LastAvailableAt),
		AvailabilityStatusError: app.AvailabilityStatusError,
		Extra:                   app.Extra,
		SourceID:                sourceId,
		ApplicationTypeID:       appTypeId,
	}
}

func (app *Application) UpdateFromRequest(req *ApplicationEditRequest) {
	if req.Extra != nil {
		// handle superkey update
		if req.Extra["_superkey"] != nil {
			// mark that we got an update (e.g. time to raise the create event)
			app.GotSuperkeyUpdate = true

			sk, _ := json.Marshal(req.Extra["_superkey"])
			app.SuperkeyData = sk

			// remove it from the hash
			delete(req.Extra, "_superkey")
		}

		b, _ := json.Marshal(req.Extra)
		app.Extra = b
	}

	if req.AvailabilityStatus != nil {
		app.AvailabilityStatus = *req.AvailabilityStatus
	}

	if req.AvailabilityStatusError != nil {
		app.AvailabilityStatusError = *req.AvailabilityStatusError
	}
}

func (app *Application) ToEmailNotificationInfo(previousStatus string) *EmailNotificationInfo {
	return &EmailNotificationInfo{
		SourceID:                   strconv.FormatInt(app.SourceID, 10),
		SourceName:                 app.Source.Name,
		ResourceDisplayName:        "Application",
		CurrentAvailabilityStatus:  app.AvailabilityStatus,
		PreviousAvailabilityStatus: previousStatus,
	}
}
