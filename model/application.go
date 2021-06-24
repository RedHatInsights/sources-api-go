package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type Application struct {
	AvailabilityStatus
	Pause
	Tenancy

	// gorm Fields
	Id        int64          `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:paused_at" json:"paused_at"`

	ApplicationTypeId       int64          `json:"application_type_id"`
	AvailabilityStatusError *string        `json:"availability_status_error,omitempty"`
	Extra                   datatypes.JSON `json:"extra,omitempty"`
	SuperkeyData            datatypes.JSON `json:"superkey_data"`

	SourceId int64 `json:"source_id"`
	Source   *Source
}
