package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lindgrenj6/sources-api-go/db"
	"github.com/lindgrenj6/sources-api-go/handlers"
	"github.com/lindgrenj6/sources-api-go/model"
)

func main() {
	db.SetupDB()
	err := db.DB.AutoMigrate(&model.Source{})
	err = db.DB.AutoMigrate(&model.Application{})

	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)

	e.GET("/sources", handlers.SourceList)
	e.GET("/sources/:id", handlers.SourceGet)
	e.POST("/sources", handlers.SourceCreate)
	e.PATCH("/sources/:id", handlers.SourceEdit)
	e.DELETE("/sources/:id", handlers.SourceDelete)

	e.Logger.Fatal(e.Start(":8000"))
}
