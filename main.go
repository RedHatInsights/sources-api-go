package main

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lindgrenj6/sources-api-go/dao"
	"github.com/lindgrenj6/sources-api-go/redis"
	"os"
)

func main() {
	dao.Init()
	redis.Init()

	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	//runMigrations()
	setupRoutes(e)
	getSourceDao = getSourceDaoWithTenant

	e.Logger.Fatal(e.Start(":8000"))
}

func runMigrations() {
	// initial import only.
	if os.Getenv("FIRST_MIGRATION") != "true" {
		tx := dao.DB.Exec("DROP TABLE schema_migrations")
		if tx.Error != nil {
			panic("failed to do initial migration.")
		}
	}

	// get the connection we already set up in the dao init
	dbConn, err := dao.DB.DB()
	if err != nil {
		panic(err)
	}
	driver, err := postgres.WithInstance(dbConn, &postgres.Config{})
	if err != nil {
		panic(err)
	}

	// create the migrator instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		panic(err)
	}

	// run migrations, panic if it fails.
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		panic(err)
	}
}
