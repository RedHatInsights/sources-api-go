package model

import (
	"fmt"
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
)

// SourceCreateRequest is a struct representing a request coming
// from the outside to create a struct, this is the way we will be marking
// fields as write-once. They are accepted on create but not edit.
type SourceCreateRequest struct {
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow string  `json:"app_creation_workflow"`
	AvailabilityStatus  string  `json:"availability_status"`

	SourceTypeID    *int64      `json:"-"`
	SourceTypeIDRaw interface{} `json:"source_type_id"`
}

// SourceEditRequest manages what we can/cannot update on the source
// object. Any extra params just will not serialize.
type SourceEditRequest struct {
	Name               *string `json:"name"`
	Version            *string `json:"version,omitempty"`
	Imported           *string `json:"imported,omitempty"`
	SourceRef          *string `json:"source_ref,omitempty"`
	AvailabilityStatus *string `json:"availability_status"`

	// TODO: remove these once satellite goes away.
	LastCheckedAt   *string `json:"last_checked_at"`
	LastAvailableAt *string `json:"last_available_at"`
}

// SourcePausedEditRequest manages the payload we allow receiving when a paused source is tried to be edited.
type SourcePausedEditRequest struct {
	AvailabilityStatus *string `json:"availability_status"`
	LastAvailableAt    *string `json:"last_available_at"`
	LastCheckedAt      *string `json:"last_checked_at"`
}

// SourceResponse represents what we will always return to the users
// of the API after a request.
type SourceResponse struct {
	AvailabilityStatus *string `json:"availability_status,omitempty"`
	LastCheckedAt      string  `json:"last_checked_at,omitempty"`
	LastAvailableAt    string  `json:"last_available_at,omitempty"`

	ID                  string  `json:"id"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	PausedAt            string  `json:"paused_at,omitempty"`
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`

	SourceTypeId string `json:"source_type_id"`
}

// SourceInternalResponse represents the structure we will return
// when a source is requested from the internal endpoint.
type SourceInternalResponse struct {
	Id                 *string `json:"id"`
	AvailabilityStatus *string `json:"availability_status"`
	ExternalTenant     *string `json:"tenant"`
	OrgId              *string `json:"org_id"`
}

func (src *Source) UpdateFromRequest(update *SourceEditRequest) {
	if update.Name != nil {
		src.Name = *update.Name
	}
	if update.Version != nil {
		src.Version = update.Version
	}
	if update.Imported != nil {
		src.Imported = update.Imported
	}
	if update.SourceRef != nil {
		src.SourceRef = update.SourceRef
	}
	if update.AvailabilityStatus != nil {
		src.AvailabilityStatus = *update.AvailabilityStatus
	}

	if update.LastAvailableAt != nil {
		t, _ := time.Parse(util.RecordDateTimeFormat, *update.LastAvailableAt)
		src.LastAvailableAt = &t
	}

	if update.LastCheckedAt != nil {
		t, _ := time.Parse(util.RecordDateTimeFormat, *update.LastCheckedAt)
		src.LastCheckedAt = &t
	}
}

func (src *Source) UpdateFromRequestPaused(update *SourcePausedEditRequest) error {
	availabilityStatus := update.AvailabilityStatus
	lastAvailableAt := update.LastAvailableAt
	lastCheckedAt := update.LastCheckedAt

	if availabilityStatus != nil {
		src.AvailabilityStatus = *availabilityStatus
	}

	if lastAvailableAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastAvailableAt)
		if err != nil {
			logging.Log.Warnf(`[source_id: %d] invalid "last available at" date received to update a paused source: %s`, src.ID, *lastAvailableAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		src.LastAvailableAt = &t
	}

	if lastCheckedAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastCheckedAt)
		if err != nil {
			logging.Log.Warnf(`[source_id: %d] invalid "last checked at" date received to update a paused source: %s`, src.ID, *lastCheckedAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		src.LastAvailableAt = &t
	}

	return nil
}
