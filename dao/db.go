package dao

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

var DB *bun.DB

func Init() {
	rawDB, err := sql.Open("pgx", dbURL())
	if err != nil {
		panic(err)
	}

	DB = bun.NewDB(rawDB, pgdialect.New())

	// This outputs the SQL bun is running in the background.
	if os.Getenv("DEBUG_SQL") == "true" {
		DB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))
	}
}

func dbURL() string {
	return "postgres://root:toor@tyranitar:5432/sources_api_development?sslmode=disable"
}
