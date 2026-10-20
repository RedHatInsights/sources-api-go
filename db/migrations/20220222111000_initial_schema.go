package migrations

import (
	_ "embed"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

//go:embed sql/1_initial_schema.sql
var schemaContents string

// InitialSchema imports the legacy schema into the current database. If the current database already has the schema
// set up, the migration will be nearly a no-op.
func InitialSchema() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220222110000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "migrate initial schema" started`)
			defer logging.Log.Info(`Migration "migrate initial schema" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.Exec(schemaContents).Error
				if err != nil {
					return err
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Exec(`DROP SCHEMA "public" CASCADE`).Error
			})

			return err
		},
	}
}
