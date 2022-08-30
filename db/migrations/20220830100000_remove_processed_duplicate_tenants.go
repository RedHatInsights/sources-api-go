package migrations

import (
	_ "embed"
	"fmt"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RemoveProcessedDuplicatedTenants removes all the tenants that were marked as duplicates after running the migration
// "RemoveDuplicatedTenantIdsOrgIds" with ID "20220608090500". It makes sure to only remove the duplicated tenants that
// don't have any related objects in any of the other tables that have an FK to the tenants table.
func RemoveProcessedDuplicatedTenants() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220830100000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "remove processed duplicate tenants" started`)
			defer logging.Log.Info(`Migration "remove processed duplicate tenants" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				// First grab the total count of the processed duplicated tenants to compare the number with the number
				// of deleted tenants.
				selectProcessedTenants := `
					SELECT
						count(t.*)
					FROM
						tenants AS t
					WHERE
						t.external_tenant LIKE 'processed-duplicate-tenant-of-%'
					OR
						t.org_id LIKE 'processed-duplicate-tenant-of-%'
				`

				var processedTenantsCount int64
				err := tx.Raw(selectProcessedTenants).Scan(&processedTenantsCount).Error
				if err != nil {
					return fmt.Errorf("unable to fetch the count of the processed duplicated tenants on the database: %s", err)
				}

				// Make sure that we are filtering by the tenant duplicates that don't have any other row which has an
				// FK to the tenants table.
				removeDuplicatedTenantsSql := `
					DELETE FROM
                        tenants AS t
                    WHERE
                        t.external_tenant LIKE 'processed-duplicate-tenant-of-%'
					OR
						t.org_id LIKE 'processed-duplicate-tenant-of-%'
                    AND
                        NOT EXISTS (SELECT 1 FROM application_authentications appAuths WHERE appAuths.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM applications AS apps WHERE apps.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM authentications AS auths WHERE auths.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM endpoints AS endp WHERE endp.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM source_rhc_connections AS srcrch WHERE srcrch.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM sources AS sources WHERE sources.tenant_id = t.id)
                    AND
                        NOT EXISTS (SELECT 1 FROM users AS usr WHERE usr.tenant_id = t.id)
				`
				result := tx.Exec(removeDuplicatedTenantsSql)
				if result.Error != nil {
					return fmt.Errorf("unable to delete the processed duplicated tenants from the database: %s", err)
				}

				logging.Log.WithFields(
					logrus.Fields{
						"total_processed_duplicated_tenants_count":   processedTenantsCount,
						"deleted_processed_duplicated_tenants_count": result.RowsAffected,
					},
				).Info("processed duplicated tenants deleted")

				return nil
			})
			return err
		},
	}
}
