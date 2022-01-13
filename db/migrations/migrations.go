package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var migrationsCollection = []*gormigrate.Migration{
	InitialSchema(),
}

// Migrate migrates the database schema to the latest version.
func Migrate(db *gorm.DB) error {
	migrateTool := gormigrate.New(db, gormigrate.DefaultOptions, migrationsCollection)
	return migrateTool.Migrate()
}
