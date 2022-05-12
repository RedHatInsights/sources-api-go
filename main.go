package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/jobs"
	logging "github.com/RedHatInsights/sources-api-go/logger"
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

	// Redis needs to be initialized first since the database uses a Redis lock to ensure that only one application at
	// a time can run the migrations.
	redis.Init()
	dao.Init()

	shutdown := make(chan struct{})
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	switch {
	case conf.StatusListener:
		go statuslistener.Run(shutdown)
	case conf.BackgroundWorker:
		go jobs.Run(shutdown)
	default:
		// launch 2 listeners - one for metrics and one for the actual application,
		// one on 8000 and one on 9000 (per clowder)
		go runServer(shutdown)
		go runMetricExporter()
	}

	// wait for a signal from the OS, gracefully terminating the echo servers
	// if/when that comes in
	s := <-interrupts
	logging.Log.Infof("Received %v, exiting", s)

	shutdown <- struct{}{}
	<-shutdown

	os.Exit(0)
}

func runServer(shutdown chan struct{}) {
	e := echo.New()
	logging.InitEchoLogger(e, conf)

	// set the binder to the one that does not allow extra parameters in payload
	e.Binder = &NoUnknownFieldsBinder{}

	// strip trailing slashes
	e.Pre(middleware.RemoveTrailingSlash())
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
	getMetaDataDao = getMetaDataDaoWithoutTenant
	getRhcConnectionDao = getDefaultRhcConnectionDao

	// hiding the ascii art to make the logs more json-like
	e.HideBanner = true
	e.HidePort = true

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}

		logging.Log.Infof("API Server started on :%v", port)

		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Warn(err)
		}
	}()

	// wait for the shutdown signal to come
	<-shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// shut down the server gracefully, with a timeout of 20 seconds
	if err := e.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		logging.Log.Fatal(err)
	}

	// let the main goroutine know we're ready to exit
	shutdown <- struct{}{}
}

func runMetricExporter() {
	// creating a separate echo router for the metrics handler - since it has to listen on a separate port.
	e := echo.New()
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// hiding the ascii art to make the logs more json-like
	e.HideBanner = true
	e.HidePort = true

	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9000"
	}

	logging.Log.Infof("Metrics Server started on :%v", port)
	e.Logger.Fatal(e.Start(":" + port))
}
