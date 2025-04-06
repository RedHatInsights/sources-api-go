package dao

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao/amazon"
	"github.com/RedHatInsights/sources-api-go/dao/vault"
	"github.com/RedHatInsights/sources-api-go/db/migrations"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB

	Vault          vault.VaultClient
	SecretsManager amazon.SecretsManagerClient

	conf = config.Get()
)

// PostgreSQL Error Codes
const (
	// PgUniqueConstraintViolation is PostgreSQL error code for unique index violation (more here: https://www.postgresql.org/docs/current/errcodes-appendix.html)
	PgUniqueConstraintViolation = "23505"
)

func Init() {
	l := &logging.GormLogger{
		SkipErrorRecordNotFound: true,
		Logger:                  logging.Log,
		SlowThreshold:           time.Duration(conf.SlowSQLThreshold) * time.Second,
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

	// per secret-store setup
	switch config.Get().SecretStore {
	case config.VaultStore:
		Vault = vault.NewClient()
	case config.SecretsManagerStore:
		SecretsManager, err = amazon.NewSecretsManagerClient()
		if err != nil {
			logging.Log.Fatal(err)
		}
	}

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
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=%s sslrootcert=%s",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		config.Get().DatabaseName,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
		config.Get().DatabaseSSLMode,
		config.Get().DatabaseCert,
	)
}

// dbStringDefaultDb returns a DSN wit the "dbname" set to "postgres", so that the connection can be used to perform
// any management operations like creating or deleting a database.
func dbStringDefaultDb() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=postgres host=%s port=%d sslmode=%s sslrootcert=%s",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
		config.Get().DatabaseSSLMode,
		config.Get().DatabaseCert,
	)
}

// openPostgresConnection connects to the maintenance "postgres" database and overrides the "DB" variable with that
// connection. A new connection needs to be opened because Postgres doesn't allow deleting the database you are
// connected too. You can search for "cannot drop the currently open database" error for more information. It is
// expected to exit the program once you've finished with the database management operations.
func openPostgresConnection(logger *logging.GormLogger) {
	db, err := gorm.Open(postgres.Open(dbStringDefaultDb()), &gorm.Config{Logger: logger})
	if err != nil {
		log.Fatalln(err)
	}

	DB = db
}
