package migrations

import (
	"fmt"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RenameForeignKeysIndexes renames the foreign keys and indexes to what Postgres names them by default.
func RenameForeignKeysIndexes() *gormigrate.Migration {
	type ApplicationAuthentications struct{}
	type ApplicationTypes struct{}
	type Applications struct{}
	type Authentications struct{}
	type Endpoints struct{}
	type RhcConnections struct{}
	type SourceRhcConnections struct{}
	type SourceTypes struct{}
	type Sources struct{}

	type renameStruct struct {
		table   interface{}
		oldName string
		newName string
	}

	indexNames := []renameStruct{
		{
			table:   &ApplicationAuthentications{},
			oldName: "index_application_authentications_on_application_id",
			newName: "application_authentications_application_id_idx",
		},
		{
			table:   &ApplicationAuthentications{},
			oldName: "index_application_authentications_on_authentication_id",
			newName: "application_authentications_authentication_id_idx",
		},
		{
			table:   &ApplicationAuthentications{},
			oldName: "index_application_authentications_on_paused_at",
			newName: "application_authentications_paused_at_idx",
		},
		{
			table:   &ApplicationAuthentications{},
			oldName: "index_application_authentications_on_tenant_id",
			newName: "application_authentications_tenant_id_idx",
		},
		{
			table:   &ApplicationAuthentications{},
			oldName: "index_on_tenant_application_authentication",
			newName: "application_authentications_tenant_id_key",
		},
		{
			table:   &ApplicationTypes{},
			oldName: "index_application_types_on_name",
			newName: "application_types_name_key",
		},
		{
			table:   &Applications{},
			oldName: "index_applications_on_application_type_id",
			newName: "applications_application_type_id_idx",
		},
		{
			table:   &Applications{},
			oldName: "index_applications_on_paused_at",
			newName: "applications_paused_at_idx",
		},
		{
			table:   &Applications{},
			oldName: "index_applications_on_source_id",
			newName: "applications_source_id_idx",
		},
		{
			table:   &Applications{},
			oldName: "index_applications_on_tenant_id",
			newName: "applications_tenant_id_idx",
		},
		{
			table:   &Authentications{},
			oldName: "index_authentications_on_paused_at",
			newName: "authentications_paused_at_idx",
		},
		{
			table:   &Authentications{},
			oldName: "index_authentications_on_resource_type_and_resource_id",
			newName: "authentications_resource_type_idx",
		},
		{
			table:   &Authentications{},
			oldName: "index_authentications_on_tenant_id",
			newName: "authentications_tenant_id_idx",
		},
		{
			table:   &Endpoints{},
			oldName: "index_endpoints_on_paused_at",
			newName: "endpoints_paused_at_idx",
		},
		{
			table:   &Endpoints{},
			oldName: "index_endpoints_on_source_id",
			newName: "endpoints_source_id_idx",
		},
		{
			table:   &Endpoints{},
			oldName: "index_endpoints_on_tenant_id",
			newName: "endpoints_tenant_id_idx",
		},
		{
			table:   &RhcConnections{},
			oldName: "index_rhc_connections_on_rhc_id",
			newName: "rhc_connections_rhc_id_idx",
		},
		{
			table:   &SourceRhcConnections{},
			oldName: "index_source_rhc_connections_on_source_id_and_rhc_connection_id",
			newName: "source_rhc_connections_source_id_idx",
		},
		{
			table:   &SourceTypes{},
			oldName: "index_source_types_on_name",
			newName: "source_types_name_key",
		},
		{
			table:   &Sources{},
			oldName: "index_sources_on_paused_at",
			newName: "sources_paused_at_idx",
		},
		{
			table:   &Sources{},
			oldName: "index_sources_on_source_type_id",
			newName: "sources_source_type_id_idx",
		},
		{
			table:   &Sources{},
			oldName: "index_sources_on_tenant_id",
			newName: "sources_tenant_id_idx",
		},
		{
			table:   &Sources{},
			oldName: "index_sources_on_uid",
			newName: "sources_uid_key",
		},
	}

	foreignKeyNames := []renameStruct{
		{
			table:   "application_authentications",
			oldName: "fk_rails_85a04922b1",
			newName: "application_authentications_tenant_id_fkey",
		},
		{
			table:   "application_authentications",
			oldName: "fk_rails_d709bbbff3",
			newName: "application_authentications_authentication_id_fkey",
		},
		{
			table:   "application_authentications",
			oldName: "fk_rails_a051188e10",
			newName: "application_authentications_application_id_fkey",
		},
		{
			table:   "applications",
			oldName: "fk_rails_ad5ea13d24",
			newName: "applications_application_type_id_fkey",
		},
		{
			table:   "applications",
			oldName: "fk_rails_cbcddd5826",
			newName: "applications_tenant_id_fkey",
		},
		{
			table:   "applications",
			oldName: "fk_rails_064e03ae58",
			newName: "applications_source_id_fkey",
		},
		{
			table:   "authentications",
			oldName: "fk_rails_28143f952b",
			newName: "authentications_tenant_id_fkey",
		},
		{
			table:   "endpoints",
			oldName: "fk_rails_430e742d27",
			newName: "endpoints_tenant_id_fkey",
		},
		{
			table:   "endpoints",
			oldName: "fk_rails_67ee0f0d63",
			newName: "endpoints_source_id_fkey",
		},
		{
			table:   "source_rhc_connections",
			oldName: "fk_rhc_connection_id",
			newName: "source_rhc_connections_rhc_connection_id_fkey",
		},
		{
			table:   "source_rhc_connections",
			oldName: "fk_source_id",
			newName: "source_rhc_connections_source_id_fkey",
		},
		{
			table:   "source_rhc_connections",
			oldName: "fk_tenant_id",
			newName: "source_rhc_connections_tenant_id_fkey",
		},
		{
			table:   "sources",
			oldName: "fk_rails_e7365b4f5b",
			newName: "sources_source_type_id_fkey",
		},
		{
			table:   "sources",
			oldName: "fk_rails_f830a376e4",
			newName: "sources_tenant_id_fkey",
		},
	}

	return &gormigrate.Migration{
		ID: "20220705103000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "rename indexes and foreign keys" started`)
			defer logging.Log.Info(`Migration "rename indexes and foreign keys" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				for _, idx := range indexNames {
					err := tx.Migrator().RenameIndex(idx.table, idx.oldName, idx.newName)
					if err != nil {
						return err
					}
				}

				for _, fk := range foreignKeyNames {
					sql := fmt.Sprintf(
						`ALTER TABLE "%s" RENAME CONSTRAINT "%s" TO "%s"`,
						fk.table,
						fk.oldName,
						fk.newName,
					)

					err := tx.Exec(sql).Error
					if err != nil {
						return err
					}
				}

				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				for _, idx := range indexNames {
					err := tx.Migrator().RenameIndex(idx.table, idx.newName, idx.oldName)
					if err != nil {
						return err
					}
				}

				for _, fk := range foreignKeyNames {
					sql := fmt.Sprintf(
						`ALTER TABLE "%s" RENAME CONSTRAINT "%s" TO "%s"`,
						fk.table,
						fk.newName,
						fk.oldName,
					)

					err := tx.Exec(sql).Error
					if err != nil {
						return err
					}
				}

				return nil
			})

			return err
		},
	}
}
