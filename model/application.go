package model

import "gorm.io/datatypes"

type Application struct {
	AvailabilityStatus
	Pause
	Tenancy
	TimeStamps

	Id                int64          `json:"id"`
	ApplicationTypeId int64          `json:"application_type_id"`
	Extra             datatypes.JSON `json:"extra,omitempty"`

	SourceId int64 `json:"source_id"`

	SuperkeyData datatypes.JSON `json:"superkey_data"`
}
