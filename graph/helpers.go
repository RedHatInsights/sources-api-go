package graph

import (
	"context"

	m "github.com/RedHatInsights/sources-api-go/model"
)

func tenantIdFromContext(ctx context.Context) *int64 {
	t := ctx.Value(m.Tenant{}).(*m.Tenant)
	return &t.Id
}
