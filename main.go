package main

import (
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func main() {
	dao.Init()
	redis.Init()

	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	setupRoutes(e)

	// setting up the DAO functions
	getSourceDao = getSourceDaoWithTenant
	getApplicationTypeDao = getApplicationTypeDaoWithoutTenant
	getSourceTypeDao = getSourceTypeDaoWithoutTenant

	e.Logger.Fatal(e.Start(":8000"))
}
