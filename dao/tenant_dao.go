package dao

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

func (t *tenantDaoImpl) GetOrCreateTenant(identity *identity.Identity) (*m.Tenant, error) {
	// Try to find the tenant. Prefer fetching it by its OrgId first, since EBS account numbers are deprecated.
	var tenant m.Tenant
	var err error

	err = DB.
		Debug().
		Model(&m.Tenant{}).
		Where("org_id = ? AND org_id != ''", identity.OrgID).
		First(&tenant).
		Error

	// If the error isn't a "Not Found" one, something went wrong.
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("unexpected error when fetching a tenant by its OrgId: %w", err)
	}

	// Try to fetch the tenant by its EBS account number otherwise.
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = DB.
			Debug().
			Model(&m.Tenant{}).
			Where("external_tenant = ? AND external_tenant != ''", identity.AccountNumber).
			First(&tenant).
			Error

		if err == nil && tenant != (m.Tenant{}) {
			logger.Log.WithFields(logrus.Fields{"account_number": identity.AccountNumber}).Warn("tenant found by its EBS account number")
		}

		// Again, if the error isn't a "Not Found" one, something went wrong.
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("unexpected error when fetching a tenant by its EBS account number: %w", err)
		}
	}

	// Looks like we didn't find the tenant: create it and return it.
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tenant.ExternalTenant = identity.AccountNumber
		tenant.OrgID = identity.OrgID

		err := DB.
			Debug().
			Create(&tenant).
			Error

		if err != nil {
			return nil, fmt.Errorf("unable to create the tenant: %w", err)
		}
	}

	return &tenant, nil
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
