package model

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type Application struct {
	AvailabilityStatus

	ID        int64      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	PausedAt  *time.Time `json:"paused_at"`

	AvailabilityStatusError string         `json:"availability_status_error,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`
	SuperkeyData            datatypes.JSON `json:"-"`
	GotSuperkeyUpdate       bool           `json:"-" gorm:"-"`

	TenantID int64
	Tenant   Tenant

	SourceID int64 `json:"source_id"`
	Source   Source

	ApplicationTypeID int64 `json:"application_type_id"`
	ApplicationType   ApplicationType

	ApplicationAuthentications []ApplicationAuthentication
}

func (app *Application) ToEvent() interface{} {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(app.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.DateTimeToRecordFormat(app.LastAvailableAt),
		LastCheckedAt:   util.DateTimeToRecordFormat(app.LastCheckedAt)}

	appEvent := &ApplicationEvent{
		AvailabilityStatusEvent: asEvent,
		PausedAt:                util.DateTimePointerToRecordFormat(app.PausedAt),
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
	asResponse := AvailabilityStatusResponse{
		AvailabilityStatus: util.StringValueOrNil(app.AvailabilityStatus.AvailabilityStatus),
		LastCheckedAt:      util.DateTimeToRFC3339(app.LastCheckedAt),
		LastAvailableAt:    util.DateTimeToRFC3339(app.LastAvailableAt),
	}

	return &ApplicationResponse{
		AvailabilityStatusResponse: asResponse,
		ID:                         id,
		CreatedAt:                  util.DateTimeToRFC3339(app.CreatedAt),
		UpdatedAt:                  util.DateTimeToRFC3339(app.UpdatedAt),
		PausedAt:                   util.DateTimePointerToRFC3339(app.PausedAt),
		AvailabilityStatusError:    app.AvailabilityStatusError,
		Extra:                      app.Extra,
		SourceID:                   sourceId,
		ApplicationTypeID:          appTypeId,
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
		app.AvailabilityStatus = AvailabilityStatus{AvailabilityStatus: *req.AvailabilityStatus}
	}

	if req.AvailabilityStatusError != nil {
		app.AvailabilityStatusError = *req.AvailabilityStatusError
	}
}
