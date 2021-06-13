package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lindgrenj6/sources-api-go/dao"
	"github.com/lindgrenj6/sources-api-go/redis"
)

func main() {
	dao.Init()
	redis.Init()

	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	setupRoutes(e)
	getSourceDao = getSourceDaoWithTenant

	e.Logger.Fatal(e.Start(":8000"))
}
