package main

import (
	"flag"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/statuslistener"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var conf = config.Get()

func main() {
	logging.InitLogger(conf)

	dao.Init()
	redis.Init()

	availabilityListener := flag.Bool("listener", false, "run availability status listener")
	flag.Parse()

	if *availabilityListener {
		statuslistener.Run()
	} else {
		runServer()
	}
}

func runServer() {
	e := echo.New()
	logging.InitEchoLogger(e, conf)

	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: logging.FormatForMiddleware(conf),
		Output: &logging.LogWriter{Output: logging.LogOutputFrom(conf.LogHandler),
			Logger:   logging.Log,
			LogLevel: conf.LogLevelForMiddlewareLogs},
	}))

	setupRoutes(e)

	// setting up the DAO functions
	getSourceDao = getSourceDaoWithTenant
	getApplicationDao = getApplicationDaoWithTenant
	getAuthenticationDao = getAuthenticationDaoWithTenant
	getApplicationAuthenticationDao = getApplicationAuthenticationDaoWithTenant
	getApplicationTypeDao = getApplicationTypeDaoWithTenant
	getSourceTypeDao = getSourceTypeDaoWithoutTenant
	getEndpointDao = getEndpointDaoWithTenant
	getMetaDataDao = getMetaDataDaoWithTenant

	// setting up the "http.Client" for the marketplace token provider
	marketplace.GetHttpClient = marketplace.GetHttpClientStdlib

	// Set up the TypeCache
	err := dao.PopulateStaticTypeCache()
	if err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Fatal(e.Start(":8000"))
}
