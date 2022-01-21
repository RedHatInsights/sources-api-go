package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

func GetOrCreateTenantID(accountNumber string) (*int64, error) {
	tenant := m.Tenant{ExternalTenant: accountNumber}

	// Find the tenant, scanning into the struct above
	result := DB.
		Where("external_tenant = ?", accountNumber).
		First(&tenant)

	// Looks like we didn't find it, create it and return the ID.
	if result.Error != nil {
		result = DB.Create(&tenant)
	}

	return &tenant.Id, result.Error
}

func TenantBy(accountNumber string) (*m.Tenant, error) {
	tenant := m.Tenant{ExternalTenant: accountNumber}

	result := DB.
		Where("external_tenant = ?", accountNumber).
		First(&tenant)

	return &tenant, result.Error
}
