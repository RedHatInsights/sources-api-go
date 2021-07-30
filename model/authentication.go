package model

type Authentication struct {
	AvailabilityStatus
	Pause
	TimeStamps

	Id int64 `json:"id"`

	Name                    *string                `json:"name,omitempty"`
	AuthType                *string                `json:"auth_type"`
	Username                *string                `json:"username"`
	Password                *string                `json:"password"`
	Extra                   map[string]interface{} `json:"extra,omitempty"`
	AvailabilityStatusError *string                `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`
	Source   Source
	TenantID int64 `json:"tenant_id"`
	Tenant   Tenant

	ResourceType string `json:"resource_type"`
	ResourceId   int64  `json:"resource_id"`
}
