package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func AddRetryCounterToApplications() *gormigrate.Migration {
	type Application struct {
		// using a smallint here, goes up to 32,000 which should be plenty.
		RetryCounter *int8 `gorm:"default:0"`
	}

	return &gormigrate.Migration{
		ID: "20220428124100",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add retry counter to applications" started`)
			defer logging.Log.Info(`Migration "add retry counter to applications" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().AddColumn(&Application{}, "RetryCounter")
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&Application{}, "RetryCounter")
			})

			return err
		},
	}
}
