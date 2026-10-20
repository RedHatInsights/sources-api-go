package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// MakeEmptyExternalTenantsOrgIdsNull makes all the empty "external_tenant" and "org_id" values NULL in the database.
func MakeEmptyExternalTenantsOrgIdsNull() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220608090000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "make empty external tenants and org ids null" started`)
			defer logging.Log.Info(`Migration "make empty external tenants and org ids null" ended`)

			err := db.Debug().Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&model.Tenant{}).
					Where("external_tenant = ''").
					Update("external_tenant", nil).
					Error

				if err != nil {
					return err
				}

				err = tx.
					Model(&model.Tenant{}).
					Where("org_id = ''").
					Update("org_id", nil).
					Error

				return err
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Debug().Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&model.Tenant{}).
					Where("external_tenant IS NULL").
					Update("external_tenant", "").
					Error

				if err != nil {
					return err
				}

				err = tx.
					Model(&model.Tenant{}).
					Where("org_id IS NULL").
					Update("org_id", "").
					Error

				return err
			})

			return err
		},
	}
}
