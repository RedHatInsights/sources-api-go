package dao

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
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
