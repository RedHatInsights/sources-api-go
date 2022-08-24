package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/RedHatInsights/tenant-utils/pkg/tenantid"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// untranslatedTenantsWhereCondition is the condition required for the "get tenants" and "update tenants" member
// functions to grab the tenants that are translatable.
const untranslatedTenantsWhereCondition = `("external_tenant" IS NOT NULL OR "external_tenant" != '') AND ("org_id" IS NULL OR "org_id" = '')`

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

		logger.Log.WithFields(
			logrus.Fields{"account_number": tenant.ExternalTenant, "org_id": tenant.OrgID, "tenant_id": tenant.Id},
		).Info("tenant created")
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

func (t *tenantDaoImpl) GetUntranslatedTenants() ([]m.Tenant, error) {
	var tenants []m.Tenant

	err := DB.Debug().
		Model(&m.Tenant{}).
		Where(untranslatedTenantsWhereCondition).
		Find(&tenants).
		Error

	if err != nil {
		return nil, err
	}

	return tenants, err
}

func (t *tenantDaoImpl) TranslateTenants() (int64, uint64, uint64, []m.TenantTranslation, error) {
	// Count the total number of translatable tenants.
	var translatableTenants int64
	// Count the translation operations.
	var translatedTenants, untranslatedTenants uint64
	// Keep track of each of the tenant translations.
	var tenantTranslations []m.TenantTranslation

	// Run everything inside a transaction to avoid inconsistencies. This way if new tenants are added/removed while
	// the translation is in progress, all the counts and the results will still match.
	err := DB.Debug().Transaction(func(tx *gorm.DB) error {
		// Get the count to be able to process the tenants in batches of 100 tenants.
		err := tx.
			Model(&m.Tenant{}).
			Where(untranslatedTenantsWhereCondition).
			Count(&translatableTenants).
			Error

		if err != nil {
			return fmt.Errorf("unable to count untranslated tenants: %w", err)
		}

		translator := tenantid.NewTranslator(config.Get().TenantTranslatorUrl)
		for i := int64(0); i < translatableTenants; i += 100 {
			// Grab all the EBS account numbers to be processed.
			var ebsAccountNumbers []string
			err := tx.
				Model(&m.Tenant{}).
				Where(untranslatedTenantsWhereCondition).
				Pluck("external_tenant", &ebsAccountNumbers).
				Offset(int(i)).
				Order(clause.OrderByColumn{Column: clause.Column{Name: "external_tenant"}, Desc: false}).
				Limit(100).
				Error

			if err != nil {
				return fmt.Errorf("unable to fetch the tenants to translate them: %w", err)
			}

			// Batch process 100 EBS account numbers.
			results, err := translator.EANsToOrgIDs(context.Background(), ebsAccountNumbers)
			if err != nil {
				return fmt.Errorf("error when calling the translation service: %w", err)
			}

			for _, result := range results {
				var tenantTranslation = m.TenantTranslation{}

				// Did the translation service return an error?
				if result.Err != nil {
					logger.Log.WithField("external_tenant", *result.EAN).Errorf(`could not translate to "org_id": %s`, err)

					tenantTranslation.ExternalTenant = *result.EAN
					tenantTranslation.Error = err
					tenantTranslations = append(tenantTranslations, tenantTranslation)

					untranslatedTenants++
					continue
				}

				// Grab the updated tenant for the "translated tenant" struct.
				var tenant m.Tenant
				// Update the tenant with the translated "orgId".
				dbResult := tx.
					Model(&tenant).
					Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
					Where("external_tenant = ?", result.EAN).
					Updates(map[string]interface{}{
						"org_id": result.OrgID,
					})

				if dbResult.Error != nil {
					translationError := fmt.Errorf(`could no translate to "org_id": %w`, dbResult.Error)

					logger.Log.WithField("external_tenant", *result.EAN).WithField("org_id", result.OrgID).Error(translationError)

					tenantTranslation.ExternalTenant = *result.EAN
					tenantTranslation.OrgId = result.OrgID
					tenantTranslations = append(tenantTranslations, tenantTranslation)

					untranslatedTenants++
					continue
				}

				// If the update yielded 0 affected rows that means that no tenant could be found with that
				// "external_tenant".
				if dbResult.RowsAffected == 0 {
					translationError := errors.New(`could not translate to "org_id": external tenant not found`)

					logger.Log.WithField("external_tenant", *result.EAN).Error(translationError)

					tenantTranslation.ExternalTenant = *result.EAN
					tenantTranslation.Error = translationError
					tenantTranslations = append(tenantTranslations, tenantTranslation)

					untranslatedTenants++
					continue
				}

				logger.Log.WithFields(
					logrus.Fields{"external_tenant": *result.EAN, "org_id": result.OrgID, "tenant_id": tenant.Id},
				).Info("successful tenant translation")

				tenantTranslation.Id = tenant.Id
				tenantTranslation.ExternalTenant = *result.EAN
				tenantTranslation.OrgId = result.OrgID
				tenantTranslations = append(tenantTranslations, tenantTranslation)

				translatedTenants++
			}
		}

		return nil
	})

	if err != nil {
		return 0, 0, 0, nil, err
	}

	return translatableTenants, translatedTenants, untranslatedTenants, tenantTranslations, nil
}
