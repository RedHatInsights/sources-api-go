package dao

import (
	"database/sql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

var DB *bun.DB

func Init() {
	rawDB, err := sql.Open("pgx", dbURL())
	if err != nil {
		panic(err)
	}

	DB = bun.NewDB(rawDB, pgdialect.New())
}

func dbURL() string {
	return "postgres://root:toor@tyranitar:5432/sources_api_development?sslmode=disable"
}
