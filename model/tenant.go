package model

import (
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
)

type Tenant struct {
	Id             int64
	ExternalTenant string
	OrgID          string
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
		{Key: h.ACCOUNT_NUMBER, Value: []byte(t.ExternalTenant)},
		{Key: h.ORGID, Value: []byte(t.OrgID)},
	}
}
