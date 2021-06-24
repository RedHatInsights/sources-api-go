package dao

import (
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	db, err := gorm.Open(postgres.Open(dbString()), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	DB = db
	rawDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	rawDB.SetMaxOpenConns(20)
}

func dbString() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		"root",
		"toor",
		"sources_api_development",
		"tyranitar",
		"5432",
	)
}
