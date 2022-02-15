package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type RhcConnection struct {
	ID    int64 `gorm:"primaryKey" json:"id"`
	RhcId string
	Extra datatypes.JSON `json:"extra,omitempty"`
	AvailabilityStatus
	AvailabilityStatusError string    `json:"availability_status_error,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	Sources []Source `gorm:"many2many:source_rhc_connections"`
}

func (r *RhcConnection) UpdateFromRequest(input *RhcConnectionUpdateRequest) {
	r.Extra = input.Extra
}

func (r *RhcConnection) ToEvent() interface{} {
	asEvent := AvailabilityStatusEvent{
		AvailabilityStatus: util.StringValueOrNil(r.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt:    util.DateTimeToRecordFormat(r.LastAvailableAt),
		LastCheckedAt:      util.DateTimeToRecordFormat(r.LastCheckedAt),
	}

	rhcConnectionEvent := &RhcConnectionEvent{
		RhcId:                   &r.RhcId,
		Extra:                   r.Extra,
		AvailabilityStatusEvent: asEvent,
		AvailabilityStatusError: &r.AvailabilityStatusError,
		CreatedAt:               util.DateTimeToRecordFormat(r.CreatedAt),
		UpdatedAt:               util.DateTimeToRecordFormat(r.UpdatedAt),
	}

	return rhcConnectionEvent
}

func (r *RhcConnection) ToResponse() *RhcConnectionResponse {
	id := strconv.FormatInt(r.ID, 10)

	var sourceIds []string
	if len(r.Sources) > 0 {
		for _, src := range r.Sources {
			sourceIds = append(sourceIds, strconv.FormatInt(src.ID, 10))
		}
	}

	return &RhcConnectionResponse{
		Id:                      &id,
		Uuid:                    &r.RhcId,
		Extra:                   r.Extra,
		AvailabilityStatus:      r.AvailabilityStatus,
		AvailabilityStatusError: r.AvailabilityStatusError,
		SourceIds:               sourceIds,
	}
}
