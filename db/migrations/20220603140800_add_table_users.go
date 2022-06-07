package migrations

import (
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func AddTableUsers() *gormigrate.Migration {
	type Tenant struct {
		Id int64
	}

	type User struct {
		Id     int64  `gorm:"primarykey"`
		UserID string `gorm:"not null; uniqueIndex:users_user_id_idx;index:users_user_id_tenant_id_idx,unique"`

		TenantID int64 `gorm:"not null; index:users_user_id_tenant_id_idx,unique"`
		Tenant   Tenant

		CreatedAt time.Time `gorm:"type: TIMESTAMP WITHOUT TIME ZONE NOT NULL"`
		UpdatedAt time.Time `gorm:"type: TIMESTAMP WITHOUT TIME ZONE NOT NULL"`
	}

	return &gormigrate.Migration{
		ID: "20220603140800",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_table_users" started`)
			defer logging.Log.Info(`Migration "add_table_users" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().CreateTable(&User{})
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&User{})
			})

			return err
		},
	}
}
