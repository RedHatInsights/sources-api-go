package model

type Application struct {
	AvailabilityStatus
	Pause
	Tenancy
	TimeStamps

	Id                int64                  `json:"id"`
	ApplicationTypeId int64                  `json:"application_type_id"`
	Extra             map[string]interface{} `json:"extra,omitempty"`

	SourceId int64 `json:"source_id"`

	SuperkeyData map[string]interface{} `json:"superkey_data"`
}
