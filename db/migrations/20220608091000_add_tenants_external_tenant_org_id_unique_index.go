package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddTenantsExternalTenantOrgIdUniqueIndex adds a unique index to the "external tenant" and "org id" columns to avoid
// a race condition that enabled having multiple rows of data sharing the same "external tenant" or "org id" values.
// This was caused by sending two parallel requests having a non-existent tenant on the database, which would create
// the two separate records containing the same values at the same time.
//
// An index is created instead of a constraint because we are going to be searching by "external_tenant" and "org_id"
// columns, so we can benefit from that index.
func AddTenantsExternalTenantOrgIdUniqueIndex() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220608091000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add a unique constraint to tenants.external_tenant column" started`)
			defer logging.Log.Info(`Migration "add a unique constraint to tenants.external_tenant column" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				createIndexes := [2]string{
					`CREATE UNIQUE INDEX "tenants_external_tenant_idx" ON "tenants"("external_tenant")`,
					`CREATE UNIQUE INDEX "tenants_org_id_idx" ON "tenants"("org_id")`,
				}

				for _, index := range createIndexes {
					err := tx.
						Debug().
						Exec(index).
						Error

					if err != nil {
						return err
					}
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				dropIndexes := [2]string{
					`DROP INDEX "tenants_external_tenant_idx"`,
					`DROP INDEX "tenants_org_id_idx"`,
				}

				for _, index := range dropIndexes {
					err := tx.
						Debug().
						Exec(index).
						Error

					if err != nil {
						return err
					}
				}

				return nil
			})

			return err
		},
	}
}
