package migrations

import (
	_ "embed"
	"fmt"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AvailabilityStatusColumnsNotNullConstraintDefaultValue adds a NOT NULL constraint to the "availability_status"
// columns from all the tables that have it, and it also adds a default value "in_progress" for new insertions. The
// migration takes care of the NULL or empty values from those columns and sets them to "in_progress".
func AvailabilityStatusColumnsNotNullConstraintDefaultValue() *gormigrate.Migration {
	// tablesToBeUpdated holds all the table names which have an "availability_status" column which may have nil or
	// empty values that need to be updated to the default value.
	tablesToBeUpdated := []string{
		"applications",
		"authentications",
		"endpoints",
		"rhc_connections",
		"sources",
	}

	return &gormigrate.Migration{
		ID: "20220905120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "remove empty or nil availability statuses" started`)
			defer logging.Log.Info(`Migration "remove empty or nil availability statuses" ended`)

			// Perform the migration.
			err := db.Transaction(func(tx *gorm.DB) error {
				for _, table := range tablesToBeUpdated {
					// Update the NULL or empty availability status values.
					err := tx.Debug().
						Table(table).
						Where(`availability_status IS NULL OR availability_status = ''`).
						Update("availability_status", "in_progress").
						Error

					if err != nil {
						return fmt.Errorf(`unable to set the NULL or empty availability statuses of table "%s" to "in_progress": %w`, table, err)
					}

					// Add a default value to the column.
					defaultSql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "availability_status" SET DEFAULT 'in_progress'`, table)
					err = tx.Debug().
						Exec(defaultSql).
						Error

					if err != nil {
						return fmt.Errorf(`unable to set a default value for the column "availability_status" of the table "%s": %w`, table, err)
					}

					// Add a NOT NULL constraint to the column.
					notNullSql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "availability_status" SET NOT NULL`, table)
					err = tx.Debug().
						Exec(notNullSql).
						Error

					if err != nil {
						return fmt.Errorf(`unable to set a NOT NULL constraint for the column "availability_status" of the table "%s": %w`, table, err)
					}
				}
				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				for _, table := range tablesToBeUpdated {
					// Drop the NOT NULL constraint.
					notNullSql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "availability_status" DROP NOT NULL`, table)

					err := tx.Debug().
						Exec(notNullSql).
						Error

					if err != nil {
						return fmt.Errorf(`unable to drop the NOT NULL constraint from the "availability_status" column of the table "%s": %w`, table, err)
					}

					// Drop the default value.
					defaultValueSql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "availability_status" DROP DEFAULT`, table)

					err = tx.Debug().
						Exec(defaultValueSql).
						Error

					if err != nil {
						return fmt.Errorf(`unable to drop the default value from the "availability_status" column of the table "%s": %w`, table, err)
					}
				}

				return nil
			})

			return err
		},
	}
}
