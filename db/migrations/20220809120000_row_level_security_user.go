package migrations

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RowLevelSecurityUser creates a new user on the database. The idea is to have a full privileged user which is the one
// who runs the migrations, and then another one which will just "use" the database. That is why this second user only
// has a limited set of permissions for each of the database's tables.
func RowLevelSecurityUser() *gormigrate.Migration {
	// Temporarily store the configuration.
	sourcesConfig := config.Get()

	user := sourcesConfig.DatabaseUserApplication
	password := sourcesConfig.DatabasePasswordApplication
	database := sourcesConfig.DatabaseName

	// Permissions is a struct that defines the set of permissions the application user will have on each table.
	permissions := []struct {
		Table              string
		Permissions        []string
		SequencePermission bool
	}{
		{
			Table:              "application_authentications",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:       "application_types",
			Permissions: []string{"SELECT"},
		},
		{
			Table:              "applications",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:              "authentications",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:              "endpoints",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:       "meta_data",
			Permissions: []string{"SELECT"},
		},
		{
			Table:              "rhc_connections",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:       "source_rhc_connections",
			Permissions: []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
		},
		{
			Table:       "source_types",
			Permissions: []string{"SELECT"},
		},
		{
			Table:              "sources",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:              "tenants",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
		{
			Table:              "users",
			Permissions:        []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
			SequencePermission: true,
		},
	}

	return &gormigrate.Migration{
		ID: "20220809120000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "create a user for row level security settings" started`)
			defer logging.Log.Info(`Migration "create a user for row level security settings" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				createUserSql := fmt.Sprintf(`CREATE USER %s WITH PASSWORD '%s'`, user, password)
				err := tx.Debug().Exec(createUserSql).Error
				if err != nil {
					return fmt.Errorf("unable to create application user for the database: %w", err)
				}

				connectUserSql := fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s`, database, user)
				err = tx.Debug().Exec(connectUserSql).Error
				if err != nil {
					return fmt.Errorf(`unable to grant the "connect to database" privilege to user "%s" on database "%s": %w`, user, database, err)
				}

				grantUsageSql := fmt.Sprintf(`GRANT USAGE ON SCHEMA "public" TO %s`, user)
				err = tx.Debug().Exec(grantUsageSql).Error
				if err != nil {
					return fmt.Errorf(`unable to grant "USAGE" permission to the low privileged user "%s": %w`, user, err)
				}

				for _, perm := range permissions {
					grantPermissionsTableSql := fmt.Sprintf(`GRANT %s ON %s TO %s`, strings.Join(perm.Permissions, ", "), perm.Table, user)
					err := tx.Debug().Exec(grantPermissionsTableSql).Error
					if err != nil {
						return fmt.Errorf(`unable to grant "%s" permissions to user "%s" on table "%s": %w`, perm.Permissions, user, perm.Table, err)
					}

					// The user must be granted access to the sequence of the table if it has an "INSERT" permission
					// granted.
					if perm.SequencePermission {
						grantSequenceSql := fmt.Sprintf(`GRANT USAGE, SELECT ON SEQUENCE %s_id_seq TO %s`, perm.Table, user)
						err := tx.Debug().Exec(grantSequenceSql).Error
						if err != nil {
							return fmt.Errorf(`unable to grant "%s" usage and select permissions on sequence "%s_id_seq" on table "%s": %w`, user, perm.Table, perm.Table, err)
						}
					}
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				for _, perm := range permissions {
					dropGrantsSql := fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s FROM %s", perm.Table, user)
					err := tx.Debug().Exec(dropGrantsSql).Error
					if err != nil {
						return fmt.Errorf(`unable to revoke all privileges on table "%s" from user "%s": %w`, perm.Table, user, err)
					}

					// Revoke the sequence permissions too from the table.
					if perm.SequencePermission {
						grantSequenceSql := fmt.Sprintf(`REVOKE USAGE, SELECT ON SEQUENCE %s_id_seq TO %s`, perm.Table, user)
						err := tx.Debug().Exec(grantSequenceSql).Error
						if err != nil {
							return fmt.Errorf(`unable to revoke "%s" usage and select permissions on sequence "%s_id_seq" on table "%s": %w`, user, perm.Table, perm.Table, err)
						}
					}
				}

				revokeUsageSchemaSql := fmt.Sprintf(`REVOKE USAGE ON SCHEMA "public" FROM %s`, user)
				err := tx.Debug().Exec(revokeUsageSchemaSql).Error
				if err != nil {
					return fmt.Errorf(`unable to revoke the usage privilege on schema "public" from user "%s": %w`, user, err)
				}

				revokeConnectDatabaseSql := fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM %s", database, user)
				err = tx.Debug().Exec(revokeConnectDatabaseSql).Error
				if err != nil {
					return fmt.Errorf(`unable to revoke the connect privilege on database "%s" to user "%s": %w`, database, user, err)
				}

				dropUserSql := fmt.Sprintf("DROP USER %s", user)
				err = tx.Debug().Exec(dropUserSql).Error
				if err != nil {
					return fmt.Errorf(`unable to drop user "%s" from database: %w`, user, err)
				}

				return nil
			})

			return err
		},
	}
}
