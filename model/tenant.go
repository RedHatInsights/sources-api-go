package model

import "time"

type Tenant struct {
	Id             int64
	ExternalTenant string
	CreatedAt      time.Time
	UpdateAt       time.Time
}
