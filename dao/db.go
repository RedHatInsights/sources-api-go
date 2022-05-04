package dao

import (
	"fmt"
	"log"
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

	// Reset the database if the command flag was provided.
	if conf.MigrationsReset {
		openPostgresConnection(l)

		// Terminate any other connections to the database, since otherwise Postgres will not allow deleting a database.
		disconnectSql := fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'`, conf.DatabaseName)
		err := DB.Exec(disconnectSql).Error
		if err != nil {
			log.Fatalln(err)
		}

		// Perform the database deletion.
		dropDbSql := fmt.Sprintf(`DROP DATABASE %s`, conf.DatabaseName)
		err = DB.Exec(dropDbSql).Error
		if err != nil {
			log.Fatalln(err)
		}

		// Recreate the database.
		createDb := fmt.Sprintf(`CREATE DATABASE %s`, conf.DatabaseName)
		err = DB.Exec(createDb).Error
		if err != nil {
			log.Fatalln(err)
		}

		// Log and exit so that the application can be rerun without the "reset" flag.
		logging.Log.Infof(`Database "%s" has been reset`, conf.DatabaseName)
		os.Exit(0)
	}

	// Create the database if the command flag was provided.
	if conf.MigrationsSetup {
		openPostgresConnection(l)

		// Using parameters on a "CREATE DATABASE" statement doesn't work, so we use Sprintf instead. This should be
		// safe since it's a command that will run.
		sql := fmt.Sprintf(`CREATE DATABASE "%s"`, conf.DatabaseName)

		err := DB.Exec(sql).Error
		if err != nil {
			logging.Log.Fatalf(`Error creating database "%s": %s`, conf.DatabaseName, err)
		}

		// Log and exit so that the application can be rerun without the "setup" flag.
		logging.Log.Infof(`Database "%s" created`, conf.DatabaseName)
		os.Exit(0)
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

	// Perform database migrations.
	migrations.Migrate(DB)

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
	if !conf.StatusListener && !conf.BackgroundWorker {
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

// dbStringDefaultDb returns a DSN wit the "dbname" set to "postgres", so that the connection can be used to perform
// any management operations like creating or deleting a database.
func dbStringDefaultDb() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=postgres host=%s port=%d sslmode=disable",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
	)
}

// openPostgresConnection connects to the maintenance "postgres" database and overrides the "DB" variable with that
// connection. A new connection needs to be opened because Postgres doesn't allow deleting the database you are
// connected too. You can search for "cannot drop the currently open database" error for more information. It is
// expected to exit the program once you've finished with the database management operations.
func openPostgresConnection(logger *logging.CustomGORMLogger) {
	db, err := gorm.Open(postgres.Open(dbStringDefaultDb()), &gorm.Config{Logger: logger})
	if err != nil {
		log.Fatalln(err)
	}

	DB = db
}
