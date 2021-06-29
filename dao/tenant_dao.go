package dao

import (
	m "github.com/lindgrenj6/sources-api-go/model"
)

func GetOrCreateTenantID(accountNumber string) *int64 {
	tenant := &m.Tenant{ExternalTenant: accountNumber}
	if result := DB.First(&tenant); result.Error != nil {
		DB.Create(&tenant)
	}

	return &tenant.Id
}
