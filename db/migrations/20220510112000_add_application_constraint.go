package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func AddApplicationConstraint() *gormigrate.Migration {
	type Application struct {
		gorm.Model
		SourceID          int64 `gorm:"uniqueIndex:applications_app_type_id_source_id_tenant_id_idx"`
		ApplicationTypeID int64 `gorm:"uniqueIndex:applications_app_type_id_source_id_tenant_id_idx"`
		TenantID          int64 `gorm:"uniqueIndex:applications_app_type_id_source_id_tenant_id_idx"`
	}

	return &gormigrate.Migration{
		ID: "20220510112000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_application_constraint" started`)
			defer logging.Log.Info(`Migration "add_application_constraint" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().
					CreateIndex(&Application{}, "applications_app_type_id_source_id_tenant_id_idx")
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().
					DropIndex(&Application{}, "applications_app_type_id_source_id_tenant_id_idx")
			})

			return err
		},
	}
}
