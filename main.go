package main

import (
	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var conf = config.Get()

func main() {
	e := echo.New()

	logging.InitLogger(conf)
	logging.InitEchoLogger(e, conf)

	dao.Init()
	redis.Init()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	setupRoutes(e)

	// setting up the DAO functions
	getSourceDao = getSourceDaoWithTenant
	getApplicationDao = getApplicationDaoWithTenant
	getApplicationTypeDao = getApplicationTypeDaoWithoutTenant
	getSourceTypeDao = getSourceTypeDaoWithoutTenant

	e.Logger.Fatal(e.Start(":8000"))
}
