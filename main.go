package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lindgrenj6/sources-api-go/db"
	"github.com/lindgrenj6/sources-api-go/model"
)

func main() {
	db.SetupDB()
	err := db.DB.AutoMigrate(&model.Source{})
	if err != nil {
		panic(err)
	}
	err = db.DB.AutoMigrate(&model.Application{})
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)

	e.Use(middleware.Logger())
	setupRoutes(e)

	e.Logger.Fatal(e.Start(":8000"))
}
