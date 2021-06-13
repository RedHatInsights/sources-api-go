package model

import "gorm.io/datatypes"

type Authentication struct {
	AvailabilityStatus
	Pause
	Tenancy
	TimeStamps

	Id int64 `json:"id"`

	Name     string         `json:"name,omitempty"`
	AuthType string         `json:"auth_type"`
	Username string         `json:"username"`
	Password string         `json:"password"`
	Extra    datatypes.JSON `json:"extra,omitempty"`

	SourceId     int64  `json:"source_id"`
	ResourceType string `json:"resource_type"`
	ResourceId   int64  `json:"resource_id"`
}
