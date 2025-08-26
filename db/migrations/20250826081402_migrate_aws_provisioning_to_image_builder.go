package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// MigrateAwsProvisioningToImageBuilder migrates AWS provisioning applications to the new Image Builder application type
func MigrateAwsProvisioningToImageBuilder() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20250826081402",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "migrate AWS provisioning applications to Image Builder" started`)
			defer logging.Log.Info(`Migration "migrate AWS provisioning applications to Image Builder" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				// Update all applications that:
				// 1. Have application_type_id matching the provisioning application type
				// 2. Are connected to sources with source_type_id matching Amazon
				query := `
					UPDATE applications 
					SET application_type_id = (
						SELECT id FROM application_types WHERE name = '/insights/platform/image-builder'
					)
					WHERE application_type_id = (
						SELECT id FROM application_types WHERE name = '/insights/platform/provisioning'
					)
					AND source_id IN (
						SELECT id FROM sources WHERE source_type_id = (
							SELECT id FROM source_types WHERE name = 'amazon'
						)
					)`

				result := tx.Exec(query)
				if result.Error != nil {
					return result.Error
				}

				logging.Log.Infof("Migrated %d AWS provisioning applications to Image Builder", result.RowsAffected)
				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				// Rollback: Move Image Builder applications back to provisioning
				// But only for AWS sources
				query := `
					UPDATE applications 
					SET application_type_id = (
						SELECT id FROM application_types WHERE name = '/insights/platform/provisioning'
					)
					WHERE application_type_id = (
						SELECT id FROM application_types WHERE name = '/insights/platform/image-builder'
					)
					AND source_id IN (
						SELECT id FROM sources WHERE source_type_id = (
							SELECT id FROM source_types WHERE name = 'amazon'
						)
					)`

				result := tx.Exec(query)
				if result.Error != nil {
					return result.Error
				}

				logging.Log.Infof("Rolled back %d Image Builder applications to AWS provisioning", result.RowsAffected)
				return nil
			})

			return err
		},
	}
} 