package migrations

import (
	"context"
	_ "embed"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/tenant-utils/pkg/tenantid"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// TranslateEbsAccountNumbersToOrgIdsSecondRun attempts to translate the current EBS account numbers to "OrgId"s.
//
// If the translation is not possible, it doesn't fail the migration: it simply logs an error. This is due to the
// transition stage we are in. Since we are proactively taking steps towards the full "OrgId" support, not having a
// translation for an EBS account number is acceptable.
//
// It also adds a comment to the "external_tenant" column, clarifying that it refers to "EBS account numbers".
//
// This migration is a second run, in an attempt to translate the EBS account numbers that are left to translate.
func TranslateEbsAccountNumbersToOrgIdsSecondRun() *gormigrate.Migration {
	type Tenant struct {
		OrgId string `gorm:"index; comment:Tenant identifier"`
	}

	return &gormigrate.Migration{
		ID: "20220404150000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "translate EBS account numbers to orgIds" started`)
			defer logging.Log.Info(`Migration "translate EBS account numbers to orgIds" ended`)

			// Get all the EBS account numbers from the database.
			var ebsAccountNumbers []string
			err := db.
				Debug().
				Model(&Tenant{}).
				Where("external_tenant IS NOT NULL").
				Pluck("external_tenant", &ebsAccountNumbers).
				Error

			if err != nil {
				return err
			}

			// Attempt to translate the EANs, unless there is nothing to translate.
			if len(ebsAccountNumbers) == 0 {
				return nil
			}

			translator := tenantid.NewTranslator(config.Get().TenantTranslatorUrl)
			results, err := translator.EANsToOrgIDs(context.Background(), ebsAccountNumbers)
			if err != nil {
				return err
			}

			for _, result := range results {
				if result.Err != nil {
					logging.Log.Errorf(`[external_tenant: %s] could not translate to "org_id": %s`, *result.EAN, err)
					continue
				}

				dbResult := db.
					Debug().
					Model(&Tenant{}).
					Where("external_tenant = ?", result.EAN).
					Updates(map[string]interface{}{
						"org_id": result.OrgID,
					})

				if dbResult.RowsAffected == 0 {
					logging.Log.Errorf(`[external_tenant: %s] could not translate to "org_id", external tenant not found`, *result.EAN)
				}

				if err != nil {
					logging.Log.Errorf(`[external_tenant: %s][org_id: %s] could no translate to "org_id": %s`, *result.EAN, result.OrgID, err)
				}
			}

			return nil
		},
		Rollback: func(db *gorm.DB) error {
			return nil
		},
	}
}
