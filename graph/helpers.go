package graph

import (
	"context"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func tenantIdFromContext(ctx context.Context) *int64 {
	t, ok := ctx.Value(m.Tenant{}).(*m.Tenant)
	if !ok {
		panic("could not pull tenant id from context")
	}
	return &t.Id
}

func getFilters(sort_by *string) []util.Filter {
	filters := make([]util.Filter, 0)
	if sort_by != nil {
		filters = append(filters, util.Filter{Operation: "sort_by", Value: []string{*sort_by}})
	}

	return filters
}
