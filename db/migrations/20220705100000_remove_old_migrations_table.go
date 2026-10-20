package migrations

import (
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RemoveOldMigrationsTable removes the ActiveRecord migrations table.
func RemoveOldMigrationsTable() *gormigrate.Migration {
	type schemaMigrations struct {
		Version string `gorm:"primaryKey;type:varchar"`
	}

	type arInternalMetadata struct {
		Key       string    `gorm:"primaryKey;type:varchar;not null"`
		Value     string    `gorm:"type:varchar"`
		CreatedAt time.Time `gorm:"type:timestamp"`
		UpdatedAt time.Time `gorm:"type:timestamp"`
	}

	return &gormigrate.Migration{
		ID: "20220705100000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "remove old migrations table" started`)
			defer logging.Log.Info(`Migration "remove old migrations table" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.Migrator().DropTable(&schemaMigrations{})
				if err != nil {
					return err
				}

				err = tx.Migrator().DropTable(&arInternalMetadata{})
				if err != nil {
					return err
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().AutoMigrate(&schemaMigrations{}, &arInternalMetadata{})
			})

			return err
		},
	}
}
