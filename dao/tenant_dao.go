package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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

func (t *tenantDaoImpl) GetOrCreateTenantID(identity *identity.Identity) (int64, error) {
	// Try to find the tenant.
	var tenant m.Tenant
	err := DB.
		Debug().
		Model(&m.Tenant{}).
		Where("org_id = ? AND org_id != ''", identity.OrgID).
		Or("external_tenant = ? AND external_tenant != ''", identity.AccountNumber).
		First(&tenant).
		Error

	// Looks like we didn't find it, create it and return the ID.
	if err != nil {
		tenant.ExternalTenant = identity.AccountNumber
		tenant.OrgID = identity.OrgID

		err := DB.
			Debug().
			Create(&tenant).
			Error

		if err != nil {
			return 0, err
		}
	}

	return tenant.Id, nil
}

func (t *tenantDaoImpl) TenantByIdentity(id *identity.Identity) (*m.Tenant, error) {
	var tenant m.Tenant

	err := DB.
		Debug().
		Model(&m.Tenant{}).
		Where("org_id = ? AND org_id != ''", id.OrgID).
		Or("external_tenant = ? AND external_tenant != ''", id.AccountNumber).
		First(&tenant).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("tenant")
	}

	return &tenant, nil
}
