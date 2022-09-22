package model

import (
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
)

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

func (t Tenant) GetHeadersWithGeneratedXRHID() []kafka.Header {
	return append(t.GetHeaders(), kafka.Header{
		Key: h.XRHID, Value: []byte(util.GeneratedXRhIdentity(t.ExternalTenant, t.OrgID)),
	})
}

func (t Tenant) GetHeaders() []kafka.Header {
	return []kafka.Header{
		{Key: h.AccountNumber, Value: []byte(t.ExternalTenant)},
		{Key: h.OrgID, Value: []byte(t.OrgID)},
	}
}

// TenantTranslation is a struct which represents a tenant translation from "EBS account number" to "OrgId".
type TenantTranslation struct {
	Id             int64  `json:"id"`
	ExternalTenant string `json:"external_tenant"`
	OrgId          string `json:"org_id"`
	Error          error  `json:"error,omitempty"`
}
