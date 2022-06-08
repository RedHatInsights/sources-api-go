package migrations

import (
	"fmt"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// duplicateTenantMark is the mark that will be left on the "external_tenant" and "org_id" columns for those tenants
// that are duplicated. The idea is to prepare the tenants to be able to apply the unique constraint on the columns
// right after.
const duplicateTenantMark = "processed-duplicate-tenant-of-%s-%d"

// RemoveDuplicatedTenantIdsOrgIds fetches all the duplicated tenants, makes sure that they are indeed duplicated, and
// unifies all the subresources in a single tenant before deleting the duplicated one.
func RemoveDuplicatedTenantIdsOrgIds() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220608090500",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "remove duplicated tenants" started`)
			defer logging.Log.Info(`Migration "remove duplicated tenants" ended`)

			// Updatable tables contains all the table names which have the "tenant_id" column which must be updated.
			updatableTables := []string{
				"application_authentications",
				"applications",
				"authentications",
				"endpoints",
				"sources",
				"source_rhc_connections",
				"users",
			}

			// This query selects all the tenants which have duplicated "external_tenant" or "org_id" column values.
			duplicatesSql := `
				SELECT
					"t"."id", "t"."name", "t"."description", "t"."external_tenant", "t"."org_id", "t"."created_at", "t"."updated_at"
				FROM
					"tenants" AS "t"
				WHERE
					"t"."external_tenant" IN (
						SELECT
							"texten"."external_tenant"
						FROM
							"tenants" AS "texten"
						GROUP BY
							"texten"."external_tenant"
						HAVING
							count("texten"."external_tenant") > 1
						AND
							"texten"."external_tenant" != ''
					)
				OR
					"t"."org_id" IN (
						SELECT
							"torgid"."org_id"
						FROM
							"tenants" AS "torgid"
						GROUP BY
							"torgid"."org_id"
						HAVING
							count("torgid"."org_id") > 1
						AND
							"torgid"."org_id" != ''
					)
			`

			// The idea is that the structure of the following maps will be as follows:
			//
			// externalTenantIds:
			// 	"12345" -> [12, 5123, 123]
			// 	"4717"	-> [47, 511, 511]
			//
			// orgIdIds:
			// 	"abcde-12345"	-> [51, 98]
			// 	"dfhea-4918"	-> [5811, 368128]
			var externalTenantIds = make(map[string][]int64)
			var orgIdIds = make(map[string][]int64)

			// The migration updates the subresources of a tenant. It first groups all the tenant IDs with their
			// duplicated "external_tenant" and "org_id". Once it has those, it loops through every table that depends
			// on the "tenants" table, and updates the "tenant_id" column of the resources from the duplicated tenants
			// to the main tenant ID. Finally, it updates the "tenants" table by marking the duplicated tenants with
			// an easily identifiable "external_tenant" and "org_id" values.
			err := db.Transaction(func(tx *gorm.DB) error {
				var tenants []model.Tenant

				err := tx.Debug().Raw(duplicatesSql).Scan(&tenants).Error
				if err != nil {
					return err
				}

				// Group all "external_tenant" and "org_id"s into the above defined maps.
				for _, t := range tenants {
					// Append the ID to the external tenants map.
					if t.ExternalTenant != "" {
						externalTenantIds[t.ExternalTenant] = append(externalTenantIds[t.ExternalTenant], t.Id)
					}

					// Append the ID to the orgIds map.
					if t.OrgID != "" {
						orgIdIds[t.OrgID] = append(orgIdIds[t.OrgID], t.Id)
					}
				}

				// mainTenantId is the main tenant ID to which the subresources will be tied to.
				var mainTenantId int64

				// Start processing the external tenants and grouping them into the main tenant.
				for externalTenant, ids := range externalTenantIds {
					// Grab the first tenant ID as the main one.
					if len(ids) > 0 {
						mainTenantId = ids[0]
					}

					// duplicateCounter will help us keep track of the duplicate count for a particular
					// "external_tenant" value.
					var duplicateCounter uint = 1

					// Since we've already grabbed the first ID from the slice, just take the rest starting from there.
					for _, id := range ids[1:] {
						// Update the old tenant ID from all the tables.
						for _, table := range updatableTables {
							err := updateTenantFromTable(tx, table, id, mainTenantId)
							if err != nil {
								return err
							}
						}

						// Mark the tenant as duplicate.
						mark := fmt.Sprintf(duplicateTenantMark, externalTenant, duplicateCounter)
						err := markExternalTenantAsDuplicate(tx, mark, id)
						if err != nil {
							return err
						}

						duplicateCounter++
					}
				}

				// Finish by processing the org IDs to group them on the main tenant. The difference with the above
				// code is that if an ID has already been processed, it will be skipped to avoid problems.
				for orgId, ids := range orgIdIds {
					// Grab the first tenant ID as the main one.
					if len(ids) > 0 {
						mainTenantId = ids[0]
					}

					// duplicateCounter will help us keep track of the duplicate count for a particular
					// "orgId" value.
					var duplicateCounter uint = 1

					// Since we've already grabbed the first ID from the slice, just take the rest starting from there.
					for _, id := range ids[1:] {
						// Update the old tenant ID from all the tables.
						for _, table := range updatableTables {
							err := updateTenantFromTable(tx, table, id, mainTenantId)
							if err != nil {
								return err
							}
						}

						// Mark the tenant as duplicate.
						mark := fmt.Sprintf(duplicateTenantMark, orgId, duplicateCounter)
						err := markOrgIdAsDuplicate(tx, mark, id)
						if err != nil {
							return err
						}

						duplicateCounter++
					}
				}

				return nil
			})

			return err
		},
		// There is nothing we can rollback in this migration. Once we have updated all the tenant IDs from the
		// subresources we would not know how to "split" again those tenant IDs into the duplicated tenants.
		Rollback: func(db *gorm.DB) error {
			return nil
		},
	}
}

// updateTenantFromTable updates the "tenant_id" column to the specified value from the given table.
func updateTenantFromTable(tx *gorm.DB, table string, oldTenantId int64, newTenantId int64) error {
	updateTableSql := `
		UPDATE
			"%s"
		SET
			"tenant_id" = ?
		WHERE
			"tenant_id" = ?
	`

	updateQuery := fmt.Sprintf(updateTableSql, table)

	err := tx.
		Debug().
		Exec(updateQuery, newTenantId, oldTenantId).
		Error

	if err != nil {
		return err
	}

	logging.Log.Infof(`Updated table "%s". Set tenant "%d" to "%d"`, table, oldTenantId, newTenantId)

	return nil
}

// markExternalTenantAsDuplicate marks the old duplicated tenant as duplicate. It modifies the "external_tenant" column
// to add a value that will let us know that these were duplicated tenants, and that they shouldn't have any related
// subresources.
func markExternalTenantAsDuplicate(tx *gorm.DB, mark string, tenantId int64) error {
	err := tx.
		Debug().
		Model(&model.Tenant{}).
		Where("id = ?", tenantId).
		Update("external_tenant", mark).
		Error

	if err != nil {
		return err
	}

	logging.Log.Infof(`Tenant "%d"'s "external_tenant" marked as processed duplicate with mark "%s"`, tenantId, mark)

	return nil
}

// markExternalTenantAsDuplicate marks the old duplicated tenant as duplicate. It modifies the "org_id" column to add a
// value that will let us know that these were duplicated tenants, and that they shouldn't have any related
// subresources.
func markOrgIdAsDuplicate(tx *gorm.DB, mark string, tenantId int64) error {
	err := tx.
		Debug().
		Model(&model.Tenant{}).
		Where("id = ?", tenantId).
		Update("org_id", mark).
		Error

	if err != nil {
		return err
	}

	logging.Log.Infof(`Tenant "%d"'s "org_id" marked as processed duplicate with mark "%s"`, tenantId, mark)

	return nil
}
