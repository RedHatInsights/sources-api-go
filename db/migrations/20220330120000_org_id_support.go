package migrations

import (
	"context"
	_ "embed"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/tenant-utils/pkg/tenantid"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// OrgIdSupport creates a new "org_id" column with an index in the "tenants" table and attempts to translate the
// current EBS account numbers to "org_id"s.
//
// If the translation is not possible, it doesn't fail the migration: it simply logs an error. This is due to the
// transition stage we are in. Since we are proactively taking steps towards the full "OrgId" support, not having a
// translation for an EBS account number is acceptable.
//
// It also adds a comment to the "external_tenant" column, clarifying that it refers to "EBS account numbers".
func OrgIdSupport() *gormigrate.Migration {
	type Tenant struct {
		OrgId string `gorm:"index; comment:Tenant identifier"`
	}

	return &gormigrate.Migration{
		ID: "20220330120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "external tenants to org ids" started`)
			defer logging.Log.Info(`Migration "external tenants to org ids" ended`)

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

			if err != nil {
				return err
			}

			// Get all the EBS account numbers from the database.
			var ebsAccountNumbers []string
			err = db.
				Debug().
				Model(&Tenant{}).
				Where("external_tenant IS NOT NULL").
				Pluck("external_tenant", &ebsAccountNumbers).
				Error

			if err != nil {
				return err
			}

			// Attempt to translate the EANs.
			translator := tenantid.NewTranslator(config.Get().TenantTranslatorUrl)
			results, err := translator.EANsToOrgIDs(context.Background(), ebsAccountNumbers)
			if err != nil {
				return err
			}

			for _, result := range results {
				if result.Err != nil {
					logging.Log.Errorf(`[external_tenant: %s] could not translate to "org_id": %s`, *result.EAN, err)
					continue
				}

				dbResult := db.
					Debug().
					Model(&Tenant{}).
					Where("external_tenant = ?", result.EAN).
					Updates(map[string]interface{}{
						"org_id": result.OrgID,
					})

				if dbResult.RowsAffected == 0 {
					logging.Log.Errorf(`[external_tenant: %s] could not translate to "org_id", external tenant not found`, *result.EAN)
				}

				if err != nil {
					logging.Log.Errorf(`[external_tenant: %s][org_id: %s] could no translate to "org_id": %s`, *result.EAN, result.OrgID, err)
				}
			}

			return nil
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
