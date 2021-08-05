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

	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: logging.FormatForMiddleware(conf),
		Skipper: func(c echo.Context) bool {
			logLevel := c.Echo().Logger.Level()
			return !logging.AllowedForMiddleware(logLevel, conf.LogLevelForMiddlewareLogs)
		},
		Output: logging.LogOutputFrom(conf.LogHandler),
	}))

	setupRoutes(e)

	// setting up the DAO functions
	getSourceDao = getSourceDaoWithTenant
	getApplicationDao = getApplicationDaoWithTenant
	getApplicationAuthenticationDao = getApplicationAuthenticationDaoWithTenant
	getApplicationTypeDao = getApplicationTypeDaoWithoutTenant
	getSourceTypeDao = getSourceTypeDaoWithoutTenant
	getEndpointDao = getEndpointDaoWithTenant
	getMetaDataDao = getMetaDataDaoWithoutTenant

	e.Logger.Fatal(e.Start(":8000"))
}
