package dao

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	m "github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var flags parser.Flags

func TestMain(t *testing.M) {
	flags = parser.ParseFlags()
	if flags.Integration {
		ConnectAndMigrateDB("dao")
	}

	code := t.Run()

	if flags.Integration {
		DropSchema("dao")
	}

	os.Exit(code)
}

// SO UHHHHHHH cyclic imports biting us again. Had to duplicate this code here
//
// luckily theoretically this is the only duplicated code place since everywhere
// else will be able to do the import fine. It's just this package that breaks
// the circular import rule.

var testDbName = "sources_api_test_go"

// ConnectToTestDB connects to the test database, populates the "dao.DB" member, and runs a schema migration.
func ConnectToTestDB(dbSchema string) {
	db, err := gorm.Open(postgres.Open(testDbString(testDbName)), &gorm.Config{NamingStrategy: schema.NamingStrategy{
		TablePrefix: dbSchema + ".",
	}})

	// Set the database's search path to the schema, so that no prefix needs to be added by default to the tables in
	// the queries.
	db.Exec(fmt.Sprintf(`SET search_path TO %s`, dbSchema))

	if err != nil {
		log.Fatalf("db must not exist - create the database '%s' first with '-createdb'. Error: %s", testDbName, err)
	}

	rawDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	rawDB.SetMaxOpenConns(20)

	// Set the dao.DB in case any tests want to use it
	DB = db
}

// CreateFixtures creates a new schema, migrates the schema and adds the required fixtures for the tests.
func CreateFixtures(schema string) {
	ConnectToTestDB(schema)
	DB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))

	MigrateSchema()

	DB.Create(&fixtures.TestTenantData)

	DB.Create(&fixtures.TestSourceTypeData)

	DB.Create(&fixtures.TestSourceData)

	DB.Create(&fixtures.TestRhcConnectionData)
	DB.Create(&fixtures.TestSourceRhcConnectionData)

	DB.Create(&fixtures.TestApplicationTypeData)
	DB.Create(&fixtures.TestApplicationData)

	UpdateTablesSequences(schema)
}

// DoneWithFixtures drops the schema and returns the "DB" object back to the "dao" schema, in case any other tests need
// the database in the previous schema.
func DoneWithFixtures(schema string) {
	DropSchema(schema)
	ConnectToTestDB("dao")
}

// DropSchema drops the database schema entirely.
func DropSchema(dbSchema string) {
	result := DB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", dbSchema))
	if result.Error != nil {
		log.Fatalf("Error in drop schema %s %s: ", dbSchema, result.Error.Error())
	}

}

// MigrateSchema migrates all the models.
func MigrateSchema() {
	err := DB.AutoMigrate(
		&m.SourceType{},
		&m.ApplicationType{},
		&m.MetaData{},

		&m.Source{},
		&m.RhcConnection{},
		&m.SourceRhcConnection{},
		&m.Application{},
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

func ConnectAndMigrateDB(packageName string) {
	ConnectToTestDB(packageName)

	if out := DB.Exec("CREATE SCHEMA IF NOT EXISTS " + packageName); out.Error != nil {
		log.Fatalf("error in creating schema " + out.Error.Error())
	}

	MigrateSchema()

	if out := DB.Exec(`SET search_path TO "$user", ` + packageName); out.Error != nil {
		log.Fatalf("error in setting schema" + out.Error.Error())
	}
}

// UpdateTablesSequences loops over all the tables from the database to update the tables' sequences to the latest id.
// When inserting data with an ID, for example `INSERT INTO mytable(id, desc) VALUES (1, "My description")`, the
// sequence doesn't get updated because an explicit ID was given. Therefore, if in the subsequent calls the ID is
// omitted, this could lead to "unique constraint violation" errors because of a duplicate id.
func UpdateTablesSequences(schema string) {
	tables := []string{
		"application_types",
		"sources",
		"source_types",
		"rhc_connections",
		"tenants",
	}

	for _, table := range tables {
		DB.Exec(fmt.Sprintf(
			`SELECT setval('%[2]s."%[1]s_id_seq"', (SELECT MAX(id) FROM "%[2]s"."%[1]s") + 1)`,
			table,
			schema,
		))
	}
}

// dateTimesAreSimilar returns true if both of the times are set on the same day. This is because the seconds, minutes
// or even hours may vary depending on the time of the day that the tests are run —it might even happen too if they
// run at 23:59:59 as well—, and even though comparing the results to a default value could suffice, this function aims
// to be a little more accurate about the comparison.
func dateTimesAreSimilar(one time.Time, other time.Time) bool {
	if one.Year() != other.Year() {
		return false
	}

	if one.Month() != other.Month() {
		return false
	}

	if one.Day() != other.Day() {
		return false
	}

	return true
}
