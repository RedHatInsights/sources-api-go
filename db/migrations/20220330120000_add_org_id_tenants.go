package migrations

import (
	_ "embed"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddOrgIdToTenants creates a new "org_id" column with an index in the "tenants" table. It also adds a comment to the
// "external_tenant" column, clarifying that it refers to "EBS account numbers".
func AddOrgIdToTenants() *gormigrate.Migration {
	type Tenant struct {
		OrgId string `gorm:"index; comment:Tenant identifier"`
	}

	return &gormigrate.Migration{
		ID: "20220330120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add orgIds to tenants" started`)
			defer logging.Log.Info(`Migration "add orgIds to tenants" ended`)

			// Perform the table migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				// Add the comment to the "external_tenant" column manually, since doing it in the struct with a "gorm"
				// tag makes "gormigrate" try to create the already existing column.
				err := tx.
					Debug().
					Exec(`COMMENT ON COLUMN "tenants"."external_tenant" IS 'EBS account number'`).
					Error

				if err != nil {
					return err
				}

				err = tx.
					Debug().
					AutoMigrate(&Tenant{})

				if err != nil {
					return err
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			// Remove the "org_id" column and remove the comment from the "external_tenant" column.
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Debug().
					Migrator().
					DropColumn(&Tenant{}, "org_id")

				if err != nil {
					return err
				}

				// Remove the comment from the "external_tenant" column manually, since doing it in the struct with a
				// "gorm" tag makes "gormigrate" try to create the already existing column.
				err = tx.
					Debug().
					Exec(`COMMENT ON COLUMN "tenants"."external_tenant" IS NULL`).
					Error

				return err
			})

			return err
		},
	}
}
