package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddEncryptionKeyIDColumn adds a nullable "encryption_key_id" column to the
// "authentications" table. The column stores the id of the keyring entry
// (see ENCRYPTION_KEYS) that encrypted the row's "password_hash_gcm" value,
// making key rotation a data change rather than a schema change: future
// rotations add a key to the keyring and re-stamp rows, with no DDL. See
// the encryption migration runbook for the full plan.
//
// NOTE: like the password_hash_gcm migration, this one is intentionally
// additive-only and is meant to be deployed ahead of the code that uses the
// column. The two column migrations are squashed into one before the branch
// goes up for review.
func AddEncryptionKeyIDColumn() *gormigrate.Migration {
	type Authentication struct {
		EncryptionKeyID *string `gorm:"column:encryption_key_id"`
	}

	return &gormigrate.Migration{
		ID: "20261119090000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_encryption_key_id_column" started`)
			defer logging.Log.Info(`Migration "add_encryption_key_id_column" ended`)

			return db.Transaction(func(tx *gorm.DB) error {
				// Nullable, no default: catalog-only in Postgres, instant at
				// any table size. Inert to binaries that do not map the field.
				return tx.Migrator().AddColumn(&Authentication{}, "EncryptionKeyID")
			})
		},
		Rollback: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_encryption_key_id_column" rollback started`)
			defer logging.Log.Info(`Migration "add_encryption_key_id_column" rollback ended`)

			return db.Transaction(func(tx *gorm.DB) error {
				// Safe to drop while keys are still in the keyring: dropping
				// it only loses routing metadata, and the rotation tool can
				// re-stamp rows (a NULL key id is treated as the active key).
				return tx.Migrator().DropColumn(&Authentication{}, "EncryptionKeyID")
			})
		},
	}
}
