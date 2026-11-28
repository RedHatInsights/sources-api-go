package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddPasswordHashGcmColumn adds a nullable "password_hash_gcm" column to the
// "authentications" table. The column holds AES-GCM ciphertext written with
// the primary encryption key, while the existing "password_hash" column keeps
// its legacy AES-CBC values until the migration window closes and it is
// dropped. See the encryption migration runbook for the full plan.
//
// NOTE: this migration is intentionally additive-only and is meant to be
// deployed ahead of, and independently from, the code that reads and writes
// the new column.
func AddPasswordHashGcmColumn() *gormigrate.Migration {
	type Authentication struct {
		PasswordGcm *string `gorm:"column:password_hash_gcm"`
	}

	return &gormigrate.Migration{
		ID: "20261118120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_password_hash_gcm_column" started`)
			defer logging.Log.Info(`Migration "add_password_hash_gcm_column" ended`)

			return db.Transaction(func(tx *gorm.DB) error {
				// Nullable, no default: catalog-only in Postgres, instant at
				// any table size. Inert to binaries that do not map the field.
				return tx.Migrator().AddColumn(&Authentication{}, "PasswordGcm")
			})
		},
		Rollback: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_password_hash_gcm_column" rollback started`)
			defer logging.Log.Info(`Migration "add_password_hash_gcm_column" rollback ended`)

			return db.Transaction(func(tx *gorm.DB) error {
				// Safe to drop while the migration window is open: the column
				// only ever holds copies of data that also lives (legacy-
				// encrypted) in "password_hash", and the backfill can re-
				// populate it from there.
				return tx.Migrator().DropColumn(&Authentication{}, "PasswordGcm")
			})
		},
	}
}
