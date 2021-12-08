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

// CreateFixtures creates the following fixtures for the database —listed in order—:
// - Tenant
// - SourceType
// - ApplicationType
// - Source
// - Application
// - Endpoint
// - MetaData
func CreateFixtures() {
	dao.DB.Create(&TestTenantData)

	dao.DB.Create(&TestSourceTypeData)
	dao.DB.Create(&TestApplicationTypeData)

	dao.DB.Create(&TestSourceData)
	dao.DB.Create(&TestApplicationData)
	dao.DB.Create(&TestEndpointData)

	dao.DB.Create(&TestMetaDataData)

	UpdateTablesSequences()
}

// CreateTestDB creates a test database. The function terminates the program with a code 0 if the creating is
// successful.
func CreateTestDB() {
	fmt.Printf("Creating database '%s'...", testDbName)
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
	tables := []string{
		"endpoints",
		"meta_data",
		"applications",
		"application_types",
		"sources",
		"source_types",
		"tenants",
	}

	for _, table := range tables {
		if result := dao.DB.Exec(fmt.Sprintf("DROP table %s", table)); result.Error != nil {
			log.Fatalf(
				"Error dropping table '%s'. Please manually delete the tables. Error: %s",
				table,
				result.Error,
			)
		}
	}
}

// MigrateSchema migrates all the models.
func MigrateSchema() {
	err := dao.DB.AutoMigrate(
		&m.Tenant{},
		
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

// UpdateTablesSequences loops over all the tables from the database to update the tables' sequences to the latest id.
// When inserting data with an ID, for example `INSERT INTO mytable(id, desc) VALUES (1, "My description")`, the
// sequence doesn't get updated because an explicit ID was given. Therefore, if in the subsequent calls the ID is
// omitted, this could lead to "unique constraint violation" errors because of a duplicate id.
func UpdateTablesSequences() {
	tables := []string{
		"endpoints",
		"meta_data",
		"applications",
		"application_types",
		"sources",
		"source_types",
		"tenants",
	}

	for _, table := range tables {
		dao.DB.Exec(fmt.Sprintf(
			"SELECT setval('%[1]s_id_seq', (SELECT MAX(id) FROM %[1]s) + 1)",
			table,
		))
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
