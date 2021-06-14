package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lindgrenj6/sources-api-go/dao"
)

func main() {
	dao.Init()

	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)

	e.Use(middleware.Logger())
	setupRoutes(e)

	e.Logger.Fatal(e.Start(":8000"))
}
