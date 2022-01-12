package dao

import (
	"fmt"
	"log"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	vault "github.com/hashicorp/vault/api"
	_ "github.com/jackc/pgx/v4/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB

	migrationsDir   = "db/migrations"
	migrationsTable = "schema_migrations_go"

	vaultClient *vault.Client
	Vault       VaultClient

	conf = config.Get()
)

func Init() {
	l := &logging.CustomGORMLogger{
		SkipErrorRecordNotFound: true,
		Logger:                  logging.Log,
		SlowThreshold:           time.Duration(conf.SlowSQLThreshold) * time.Second,
		LogLevelForSqlLogs:      conf.LogLevelForSqlLogs,
	}

	// Open up the conn to postgres
	db, err := gorm.Open(postgres.Open(dbString()), &gorm.Config{Logger: l})
	if err != nil {
		panic(err)
	}
	DB = db
	rawDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	rawDB.SetMaxOpenConns(20)

	// Perform database migrations
	err = migrateDatabase(db)
	if err != nil {
		log.Fatal(err)
	}

	// Open up the conn to Vault
	cfg := vault.DefaultConfig()
	if cfg == nil {
		panic("Failed to parse Vault Config")
	}
	err = cfg.ReadEnvironment()
	if err != nil {
		panic(fmt.Sprintf("Failed to read Vault Environment: %v", err))
	}

	vaultClient, err = vault.NewClient(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to Create Vault Client: %v", err))
	}

	Vault = vaultClient.Logical()

	// we only want to seed the database when running the api pod - not the status listener
	if !conf.StatusListener {
		err = seedDatabase()
		if err != nil {
			logging.Log.Fatalf("Failed to seed db: %v", err)
		}
	}

	// Set up the TypeCache
	err = PopulateStaticTypeCache()
	if err != nil {
		logging.Log.Fatalf("Failed to populate static type cache: %v", err)
	}
}

// migrateDatabase migrates the database to the latest schema version.
func migrateDatabase(db *gorm.DB) error {
	// Extract the connection from Gorm to use it for the migrations
	connection, err := db.DB()
	if err != nil {
		return err
	}

	// Configure a different migrations table, so it doesn't clash with the already existing one
	migrationsConfig := &migratePostgres.Config{MigrationsTable: migrationsTable}
	driver, err := migratePostgres.WithInstance(connection, migrationsConfig)
	if err != nil {
		return err
	}

	// Get the migrations client and specify the migrations directory
	migrateClient, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsDir,
		config.Get().DatabaseName,
		driver,
	)

	if err != nil {
		return err
	}

	// Migrate the schema
	if err := migrateClient.Up(); err != nil {
		return err
	}

	return nil
}

func dbString() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		config.Get().DatabaseName,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
	)
}
