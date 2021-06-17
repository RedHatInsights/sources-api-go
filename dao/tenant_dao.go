package dao

import (
	"time"

	m "github.com/lindgrenj6/sources-api-go/model"
)

func GetOrCreateTenantID(accountNumber string) *int64 {
	tenant := &m.Tenant{}
	err := DB.NewSelect().
		Model(tenant).
		Where("external_tenant = ?", accountNumber).
		Limit(1).
		Scan(ctx, tenant)

	// didn't find a tenant - let's create one!
	if err != nil {
		var id int64
		var now = time.Now()
		tenant = &m.Tenant{
			ExternalTenant: accountNumber,
			TimeStamps:     m.TimeStamps{CreatedAt: &now, UpdatedAt: &now},
		}
		_, err = DB.NewInsert().
			Model(tenant).
			Returning("id").
			Exec(ctx, &id)

		return &id
	}

	return &tenant.Id
}
