package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// CleanupProvisioningAuthentications removes dangling authentications with provisioning authentication types
func CleanupProvisioningAuthentications() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20250827090553",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "cleanup provisioning authentications" started`)
			defer logging.Log.Info(`Migration "cleanup provisioning authentications" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				// Delete all authentications with provisioning authentication types
				// These types are no longer supported by any application type
				provisioningAuthTypes := []string{
					"provisioning-arn",
					"provisioning_lighthouse_subscription_id", 
					"provisioning_project_id",
				}

				query := `DELETE FROM authentications WHERE authtype = ANY($1)`
				result := tx.Exec(query, provisioningAuthTypes)
				if result.Error != nil {
					return result.Error
				}

				logging.Log.Infof("Deleted %d provisioning authentications", result.RowsAffected)
				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			logging.Log.Warn("Rollback for cleanup provisioning authentications is not possible - deleted data cannot be restored")
			return nil
		},
	}
} 