package model

import "time"

// Tenant represents the tenant object we will store on the database. The "external_tenant" and "org_id" columns have a
// "null" default value, so that Gorm inserts "null" when receiving an empty "AccountNumber" or "OrgId" value from an
// identity header. Empty values are considered when enforcing the "unique index" on those columns, whilst the "NULL"s
// are not considered.
type Tenant struct {
	Id             int64
	ExternalTenant string `gorm:"default:null"`
	OrgID          string `gorm:"default:null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
