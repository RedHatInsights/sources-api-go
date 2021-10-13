package testutils

import (
	"fmt"
	"os"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/gommon/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testDbName = "sources_api_test_go"

// ConnectToTestDB connects to the test database, populates the "dao.DB" member, and runs a schema migration.
func ConnectToTestDB() {
	db, err := gorm.Open(postgres.Open(testDbString(testDbName)), &gorm.Config{})
	if err != nil {
		log.Fatalf("db must not exist - create the database '%s' first with '-createdb'. Error: %s", testDbName, err)
	}

	rawDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	rawDB.SetMaxOpenConns(20)

	// Set the dao.DB in case any tests want to use it
	dao.DB = db
	// Migrate the schema for the first time
	MigrateSchema()
}

// CreateTestDB creates a test database. The function terminates the program with a code 0 if the creating is
// successful.
func CreateTestDB() {
	db, err := gorm.Open(postgres.Open(testDbString("postgres")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error opening the database connection: %s", err)
	}

	if out := db.Exec(fmt.Sprintf("CREATE DATABASE %v", testDbName)); out.Error != nil {
		log.Fatalf("Error creating the test database: %s", out.Error)
	}

	os.Exit(0)
}

// DropSchema drops the database schema entirely.
func DropSchema() {
	dao.DB.Exec("DROP TABLE endpoints")
	dao.DB.Exec("DROP TABLE meta_data")
	dao.DB.Exec("DROP TABLE applications")
	dao.DB.Exec("DROP TABLE application_types")
	dao.DB.Exec("DROP TABLE sources")
	dao.DB.Exec("DROP TABLE source_types")
	dao.DB.Exec("DROP TABLE tenants")
}

// MigrateSchema migrates all the models.
func MigrateSchema() {
	// migrate all the models.
	err := dao.DB.AutoMigrate(
		&m.SourceType{},
		&m.ApplicationType{},

		&m.Source{},
		&m.Application{},

		&m.Endpoint{},
		&m.MetaData{},
	)

	if err != nil {
		log.Fatalf("Error automigrating the schema: %s", err)
	}
}

// testDbString returns a properly formatted database string ready to be passed to Gorm.
func testDbString(dbname string) string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		dbname,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
	)
}
