package migrations

import (
	_ "embed"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// SourceTypesAddCategoryColumn adds a "category" column to the "source_types" table. The column will have a "not null"
// constraint, so for that the migration performs three steps inside the transaction:
//
// 1. Creates the column with the not null constraint and defaults it to "Cloud".
// 2. Loops through all the source types and updates them with their corresponding category name.
// 3. Removes the default value for the column.
func SourceTypesAddCategoryColumn() *gormigrate.Migration {
	type SourceType struct {
		Category string `gorm:"column:category;comment:Category of the source;default:Cloud;not null;"`
	}

	return &gormigrate.Migration{
		ID: "20220413120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "source types: add category column" started`)
			defer logging.Log.Info(`Migration "source types: add category column" ended`)

			err := db.Debug().Transaction(func(tx *gorm.DB) error {
				// Create the new column.
				err := tx.AutoMigrate(&SourceType{})
				if err != nil {
					return err
				}

				// Find all the source types to be updated.
				var dbSourceTypes []model.SourceType
				err = tx.
					Model(&SourceType{}).
					Find(&dbSourceTypes).
					Error

				if err != nil {
					return err
				}

				// Update the category column for each source type.
				for _, sourceType := range dbSourceTypes {
					switch sourceType.Name {
					case "amazon":
						sourceType.Category = model.CategoryCloud
					case "ansible-tower":
						sourceType.Category = model.CategoryRedhat
					case "azure":
						sourceType.Category = model.CategoryCloud
					case "bitbucket":
						sourceType.Category = model.CategoryDeveloperSources
					case "dockerhub":
						sourceType.Category = model.CategoryDeveloperSources
					case "github":
						sourceType.Category = model.CategoryDeveloperSources
					case "gitlab":
						sourceType.Category = model.CategoryDeveloperSources
					case "google":
						sourceType.Category = model.CategoryCloud
					case "ibm":
						sourceType.Category = model.CategoryCloud
					case "openshift":
						sourceType.Category = model.CategoryRedhat
					case "oracle-cloud-infrastructure":
						sourceType.Category = model.CategoryCloud
					case "quay":
						sourceType.Category = model.CategoryDeveloperSources
					case "rh-marketplace":
						sourceType.Category = model.CategoryRedhat
					case "satellite":
						sourceType.Category = model.CategoryRedhat
					}

					err = tx.
						Updates(sourceType).
						Error

					if err != nil {
						return err
					}
				}

				// Remove the default value for the column.
				err = tx.
					Exec(`ALTER TABLE "source_types" ALTER COLUMN "category" DROP DEFAULT`).
					Error

				return err
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&SourceType{}, "category")
			})

			return err
		},
	}
}
