package dao

import (
	"fmt"
	"os"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/db/migrations"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	vault "github.com/hashicorp/vault/api"
	_ "github.com/jackc/pgx/v4/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB

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
	if conf.MigrationsReset {
		err = DB.Exec(`DROP SCHEMA "public" CASCADE`).Error
		if err != nil {
			logging.Log.Fatalf(`Error dropping the "public" schema: %s`, err)
		}

		err = DB.Exec(`CREATE SCHEMA "public"`).Error
		if err != nil {
			logging.Log.Fatalf(`Error creatings the "public" schema: %s`, err)
		}
	}

	if conf.MigrationsSetup {
		// Using parameters on a "CREATE DATABASE" statement doesn't work, so we use Sprintf instead. This should be
		// safe since it's a command that will run.
		sql := fmt.Sprintf(`CREATE DATABASE "%s"`, conf.DatabaseName)

		err = DB.Exec(sql).Error
		if err != nil {
			logging.Log.Fatalf(`Error creating database "%s": %s`, conf.DatabaseName, err)
		}

		// Log and exit so that the application can be rerun without the "setup" flag.
		logging.Log.Infof(`Database "%s" created`, conf.DatabaseName)
		os.Exit(0)
	}

	err = migrations.Migrate(DB)
	if err != nil {
		logging.Log.Fatalf(`Error migrating database "%s": %s`, conf.DatabaseName, err)
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
