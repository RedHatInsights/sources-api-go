package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type RhcConnection struct {
	ID    int64          `gorm:"primaryKey" json:"id"`
	RhcId string         `json:"rhc_id"`
	Extra datatypes.JSON `json:"extra,omitempty"`

	AvailabilityStatus      string     `json:"availability_status,omitempty"`
	LastCheckedAt           *time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt         *time.Time `json:"last_available_at,omitempty"`
	AvailabilityStatusError string     `json:"availability_status_error,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`

	Sources []Source `gorm:"many2many:source_rhc_connections"`
}

func (r *RhcConnection) UpdateFromRequest(input *RhcConnectionEditRequest) {
	r.Extra = input.Extra
}

func (r *RhcConnection) ToEvent() interface{} {
	id := strconv.FormatInt(r.ID, 10)

	rhcConnectionEvent := &RhcConnectionEvent{
		ID:                      &id,
		RhcId:                   &r.RhcId,
		Extra:                   r.Extra,
		AvailabilityStatus:      util.StringValueOrNil(r.AvailabilityStatus),
		LastAvailableAt:         util.DateTimePointerToRecordFormat(r.LastAvailableAt),
		LastCheckedAt:           util.DateTimePointerToRecordFormat(r.LastCheckedAt),
		AvailabilityStatusError: &r.AvailabilityStatusError,
		CreatedAt:               util.DateTimeToRecordFormat(r.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(r.UpdatedAt),
	}

	return rhcConnectionEvent
}

func (r *RhcConnection) ToResponse() *RhcConnectionResponse {
	id := strconv.FormatInt(r.ID, 10)

	sourceIds := make([]string, len(r.Sources))
	for i, src := range r.Sources {
		sourceIds[i] = strconv.FormatInt(src.ID, 10)
	}

	return &RhcConnectionResponse{
		Id:                      &id,
		RhcId:                   &r.RhcId,
		Extra:                   r.Extra,
		AvailabilityStatus:      r.AvailabilityStatus,
		AvailabilityStatusError: r.AvailabilityStatusError,
		SourceIds:               sourceIds,
	}
}

func (r *RhcConnection) ToEmail(previousStatus string) *EmailNotificationInfo {
	return &EmailNotificationInfo{
		ResourceDisplayName:        "RHC Connection",
		CurrentAvailabilityStatus:  r.AvailabilityStatus,
		PreviousAvailabilityStatus: previousStatus,
	}
}
