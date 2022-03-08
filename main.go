package main

import (
	"flag"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/statuslistener"
	echoMetrics "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// set the binder to the one that does not allow extra parameters in payload
	e.Binder = &NoUnknownFieldsBinder{}

	// recover from any `panic()`'s that happen in the handler, so the server doesn't crash.
	e.Use(middleware.Recover())
	// set up logging with our custom logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: logging.FormatForMiddleware(conf),
		Output: &logging.LogWriter{Output: logging.LogOutputFrom(conf.LogHandler),
			Logger:   logging.Log,
			LogLevel: conf.LogLevelForMiddlewareLogs},
	}))

	// use the echo prometheus middleware - without having it mount the route on the main listener.
	p := echoMetrics.NewPrometheus("sources", nil)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc { return p.HandlerFunc(next) })

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
	getRhcConnectionDao = getDefaultRhcConnectionDao

	// Set up marketplace's token management functions
	dao.GetMarketplaceTokenCacher = dao.GetMarketplaceTokenCacherWithTenantId
	dao.GetMarketplaceTokenProvider = dao.GetMarketplaceTokenProviderWithApiKey

	// setting up the "http.Client" for the marketplace token provider
	marketplace.GetHttpClient = marketplace.GetHttpClientStdlib

	// Set up the TypeCache
	err := dao.PopulateStaticTypeCache()
	if err != nil {
		e.Logger.Fatal(err)
	}

	// launch 2 listeners - one for metrics and one for the actual application
	go metricExporter()
	e.HideBanner = true
	e.Logger.Fatal(e.Start(":8000"))
}

func metricExporter() {
	// creating a separate echo router for the metrics handler - since it has to listen on a separate port.
	e := echo.New()
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	logging.Log.Infof("Metrics started on :%d", config.Get().MetricsPort)

	// we need to have 2 listeners, one on 8000 and one on 9000 (per clowder)
	e.HideBanner = true
	e.Logger.Fatal(e.Start(":9000"))
}
