package model

import "time"

type Tenant struct {
	Id             int64
	ExternalTenant string
	OrgID          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
