package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddTenantsExternalTenantOrgIdUnique adds a unique constraint to the "external tenant" and "org id" columns to avoid
// a race condition that enabled having multiple rows of data sharing the same "external tenant" or "org id" values.
// This was caused by sending two parallel requests having a non-existent tenant on the database, which would create
// the two separate records containing the same values at the same time.
func AddTenantsExternalTenantOrgIdUnique() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220504080000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add a unique constraint to tenants.external_tenant column" started`)
			defer logging.Log.Info(`Migration "add a unique constraint to tenants.external_tenant column" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				addConstraints := [2]string{
					`ALTER TABLE "tenants" ADD CONSTRAINT "tenants_external_tenant_key" UNIQUE ("external_tenant")`,
					`ALTER TABLE "tenants" ADD CONSTRAINT "tenants_org_id_key" UNIQUE ("org_id")`,
				}

				for _, constraint := range addConstraints {
					err := tx.
						Debug().
						Exec(constraint).
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
				dropConstraints := [2]string{
					`ALTER TABLE "tenants" DROP CONSTRAINT "tenants_external_tenant_key"`,
					`ALTER TABLE "tenants" DROP CONSTRAINT "tenants_org_id_key"`,
				}

				for _, constraint := range dropConstraints {
					err := tx.
						Debug().
						Exec(constraint).
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
