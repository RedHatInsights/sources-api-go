package dao

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/db/migrations"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var flags parser.Flags

// mainSchema holds the default schema to which the connection will switch when switching schemas.
var defaultSchema = "dao"

func TestMain(t *testing.M) {
	flags = parser.ParseFlags()

	logging.Log = logrus.New()

	if flags.Integration {
		Vault = &mocks.MockVault{}
		ConnectAndMigrateDB("dao")
	}

	logging.InitLogger(config.Get())
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

// CloseConnection closes the connection to the database. Useful to avoid the "max connections reached" error due to
// the many database tests we have.
func CloseConnection() {
	connection, err := DB.DB()
	if err != nil {
		log.Fatalf(`could not get the database connection: %s`, err)
	}

	err = connection.Close()
	if err != nil {
		log.Fatalf(`could not close the connection to the database: %s`, err)
	}
}

// ConnectAndMigrateDB connects to the database, switches to the provided schema, migrates the schema and creates the
// fixtures for it.
func ConnectAndMigrateDB(schema string) {
	ConnectToTestDB(schema)
	MigrateSchema()
	CreateFixtures(defaultSchema)
}

// ConnectToTestDB connects to the test database, populates the "dao.DB" member, and runs a schema migration.
func ConnectToTestDB(dbSchema string) {
	db, err := gorm.Open(postgres.Open(testDbString(testDbName)), &gorm.Config{NamingStrategy: schema.NamingStrategy{
		TablePrefix: dbSchema + ".",
	}})

	if err != nil {
		log.Fatalf(`could not connect to database: %s`, err)
	}

	rawDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	rawDB.SetMaxOpenConns(20)

	// Set the dao.DB in case any tests want to use it
	DB = db

	err = DB.
		Debug().
		Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, dbSchema)).
		Error

	if err != nil {
		log.Fatalf(`could not create schema "%s": %s`, dbSchema, err)
	}

	// Set the database's search path to the schema, so that no prefix needs to be added by default to the tables in
	// the queries.
	err = DB.
		Debug().
		Exec(fmt.Sprintf(`SET search_path TO %s`, dbSchema)).
		Error

	if err != nil {
		log.Fatalf(`could not set search path to "%s": %s`, dbSchema, err)
	}
}

// CreateFixtures creates a new schema, migrates the schema and adds the required fixtures for the tests.
func CreateFixtures(schema string) {
	DB.Create(&fixtures.TestTenantData)

	DB.Create(&fixtures.TestSourceTypeData)

	DB.Create(&fixtures.TestSourceData)
	DB.Create(&fixtures.TestEndpointData)

	DB.Create(&fixtures.TestRhcConnectionData)
	DB.Create(&fixtures.TestSourceRhcConnectionData)

	DB.Create(&fixtures.TestApplicationTypeData)
	DB.Create(&fixtures.TestApplicationData)

	DB.Create(&fixtures.TestAuthenticationData)
	DB.Create(&fixtures.TestApplicationAuthenticationData)

	DB.Create(&fixtures.TestMetaDataData)

	UpdateTablesSequences(schema)
}

// DropSchema drops the database schema entirely.
func DropSchema(dbSchema string) {
	err := DB.
		Debug().
		Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", dbSchema)).
		Error

	if err != nil {
		log.Fatalf(`Error when dropping the schema "%s": %s`, dbSchema, err)
	}
}

// MigrateSchema migrates all the models for the current schema.
func MigrateSchema() {
	// Perform the migrations and store the error for a proper return.
	migrateTool := gormigrate.New(DB, gormigrate.DefaultOptions, migrations.MigrationsCollection)
	err := migrateTool.Migrate()

	if err != nil {
		log.Fatalf(`error migrating the schema: %s`, err)
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

// SwitchSchema switches the schema to the specified one, migrates the schema and creates the fixtures for it.
func SwitchSchema(schema string) {
	CloseConnection()
	ConnectToTestDB(schema)

	err := DB.
		Debug().
		Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schema)).
		Error

	if err != nil {
		log.Fatalf(`could not create schema "%s": %s`, schema, err)
	}

	// Set the database's search path to the schema, so that no prefix needs to be added by default to the tables in
	// the queries.
	err = DB.
		Debug().
		Exec(fmt.Sprintf(`SET search_path TO %s`, schema)).
		Error

	if err != nil {
		log.Fatalf(`could not switch schema to "%s": %s`, schema, err)
	}

	MigrateSchema()
	CreateFixtures(schema)
}

// UpdateTablesSequences loops over all the tables from the database to update the tables' sequences to the latest id.
// When inserting data with an ID, for example `INSERT INTO mytable(id, desc) VALUES (1, "My description")`, the
// sequence doesn't get updated because an explicit ID was given. Therefore, if in the subsequent calls the ID is
// omitted, this could lead to "unique constraint violation" errors because of a duplicate id.
func UpdateTablesSequences(schema string) {
	tables := []string{
		"applications",
		"application_types",
		"sources",
		"source_types",
		"rhc_connections",
		"tenants",
		"users",
		"applications",
		"endpoints",
		"rhc_connections",
		"authentications",
		"application_authentications",
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
