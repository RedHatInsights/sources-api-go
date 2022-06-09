package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddResouceOwnershipToApplicationTypes adds new column "resource_ownership" into table
// "application_types".
func AddResouceOwnershipToApplicationTypes() *gormigrate.Migration {
	type ApplicationType struct {
		ResourceOwnership *string `gorm:"type:CHARACTER VARYING"`
	}

	return &gormigrate.Migration{
		ID: "20220607215900",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_resource_ownership_column" started`)
			defer logging.Log.Info(`Migration "add_resource_ownership_column" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().AddColumn(&ApplicationType{}, "ResourceOwnership")
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&ApplicationType{}, "ResourceOwnership")
			})

			return err
		},
	}
}
