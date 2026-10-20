package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddUserIdColumnIntoTables adds into tables "sources", "applications", "authentications"
// and "application_authentications" a new column "user_id" and creates in each table
// a foreign key, e.g. "sources.user_id" -> "users.id".
func AddUserIdColumnIntoTables() *gormigrate.Migration {
	type User struct {
		ID int64
	}

	type Source struct {
		UserID *int64
		User   User
	}

	type Application struct {
		UserID *int64
		User   User
	}

	type ApplicationAuthentication struct {
		UserID *int64
		User   User
	}

	type Authentication struct {
		UserID *int64
		User   User
	}

	return &gormigrate.Migration{
		ID: "20220607140800",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add_user_id_column" started`)
			defer logging.Log.Info(`Migration "add_user_id_column" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				//Add column "user_id" into "sources" table
				err := tx.Migrator().AddColumn(&Source{}, "UserID")
				if err != nil {
					return err
				}
				// Add foreign key between sources.user_id and users_id
				err = tx.Migrator().CreateConstraint(&Source{}, "User")
				if err != nil {
					return err
				}

				err = tx.Migrator().AddColumn(&Application{}, "UserID")
				if err != nil {
					return err
				}
				err = tx.Migrator().CreateConstraint(&Application{}, "User")
				if err != nil {
					return err
				}

				err = tx.Migrator().AddColumn(&ApplicationAuthentication{}, "UserID")
				if err != nil {
					return err
				}
				err = tx.Migrator().CreateConstraint(&ApplicationAuthentication{}, "User")
				if err != nil {
					return err
				}

				err = tx.Migrator().AddColumn(&Authentication{}, "UserID")
				if err != nil {
					return err
				}
				err = tx.Migrator().CreateConstraint(&Authentication{}, "User")
				if err != nil {
					return err
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.Migrator().DropConstraint(&Source{}, "User")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Source{}, "UserID")
				if err != nil {
					return err
				}

				err = tx.Migrator().DropConstraint(&Application{}, "User")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Application{}, "UserID")
				if err != nil {
					return err
				}

				err = tx.Migrator().DropConstraint(&ApplicationAuthentication{}, "User")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&ApplicationAuthentication{}, "UserID")
				if err != nil {
					return err
				}

				err = tx.Migrator().DropConstraint(&Authentication{}, "User")
				if err != nil {
					return err
				}
				err = tx.Migrator().DropColumn(&Authentication{}, "UserID")
				if err != nil {
					return err
				}

				return nil
			})

			return err
		},
	}
}
