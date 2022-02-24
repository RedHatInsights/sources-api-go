package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

// GetTenantDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetTenantDao func() TenantDao

// getDefaultRhcConnectionDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultTenantDao() TenantDao {
	return &tenantDaoImpl{}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetTenantDao = getDefaultTenantDao
}

type tenantDaoImpl struct{}

func (t *tenantDaoImpl) GetOrCreateTenantID(accountNumber string) (*int64, error) {
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

func (t *tenantDaoImpl) TenantByAccountNumber(accountNumber string) (*m.Tenant, error) {
	tenant := m.Tenant{ExternalTenant: accountNumber}

	result := DB.
		Where("external_tenant = ?", accountNumber).
		First(&tenant)

	return &tenant, result.Error
}
