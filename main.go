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
	"github.com/RedHatInsights/sources-api-go/kessel"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/statuslistener"
	"github.com/RedHatInsights/sources-api-go/util"
	echoUtils "github.com/RedHatInsights/sources-api-go/util/echo"
	echoMetrics "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var conf = config.Get()

func main() {
	// Initialize the encryption manually.
	//
	// Previously, an "init" function was used for this in the "encryption.go" file. The problem with that is that the
	// Superkey worker depends on Sources API Go, and the "InitializeEncryption" function makes a call to the
	// "setDefaultEncryptionKey" function, which in turn makes a call load the Sources API Go configuration. When the
	// Superkey runs in stage or production, the configuration that it has is obviously different to the one that the
	// Sources API Go has, which causes panics when the "setDefaultEncryptionKey" attempts to load the configuration.
	util.InitializeEncryption()

	l.InitLogger(conf)

	// Redis needs to be initialized first since the database uses a Redis lock to ensure that only one application at
	// a time can run the migrations.
	redis.Init()
	dao.Init()

	//Initializing Kessel
	var opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient("localhost:9000", opts...)
	if err != nil {
		l.Log.Fatalf("Unable to connect to Kessel: %s", err)
	}
	kesselAuthorization := kessel.NewKesselAuthorizationService(conn)
	kesselAuthorization.HasPermissionOnWorkspace(context.Background(), "3c3ef849-d8ca-11ef-ad01-083a885cd988", "12345")

	shutdown := make(chan struct{})
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	// Run the metrics exporter regardless of the application we are launching. This ensures that Prometheus is able
	// to scrape some data do produce the "up" metric. More information about it here: https://issues.redhat.com/browse/RHCLOUD-38530.
	go runMetricExporter()

	switch {
	case conf.StatusListener:
		go statuslistener.Run(shutdown)
	case conf.BackgroundWorker:
		go jobs.Run(shutdown)
	default:
		// launch 2 listeners - one for metrics and one for the actual application,
		// one on 8000 and one on 9000 (per clowder)
		go runServer(shutdown, kesselAuthorization)
		go runMetricExporter()
	}
	l.Log.Info(conf)
	// wait for a signal from the OS, gracefully terminating the echo servers
	// if/when that comes in
	s := <-interrupts
	l.Log.Infof("Received %v, exiting", s)

	shutdown <- struct{}{}
	<-shutdown

	os.Exit(0)
}

func runServer(shutdown chan struct{}, kesselService kessel.KesselAuthorizationService) {
	e := echo.New()

	// set the logger to the wrapper of our main logrus logger, with no fields on it.
	e.Logger = l.EchoLogger{Entry: l.Log.WithFields(logrus.Fields{})}

	// set the binder to the one that does not allow extra parameters in payload
	e.Binder = &echoUtils.NoUnknownFieldsBinder{}

	// strip trailing slashes
	e.Pre(middleware.RemoveTrailingSlash())
	// recover from any `panic()`'s that happen in the handler, so the server doesn't crash.
	e.Use(middleware.Recover())

	// use the echo prometheus middleware - without having it mount the route on the main listener.
	p := echoMetrics.NewPrometheus("sources", nil)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc { return p.HandlerFunc(next) })

	setupRoutes(e, kesselService)

	// setting up the DAO functions
	getSourceDao = getSourceDaoWithTenant
	getApplicationDao = getApplicationDaoWithTenant
	getAuthenticationDao = getAuthenticationDaoWithTenant
	getApplicationAuthenticationDao = getApplicationAuthenticationDaoWithTenant
	getApplicationTypeDao = getApplicationTypeDaoWithTenant
	getSecretDao = getSecretDaoWithTenant
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

		l.Log.Infof("API Server started on :%v", port)

		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			l.Log.Warn(err)
		}
	}()

	// wait for the shutdown signal to come
	<-shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// shut down the server gracefully, with a timeout of 20 seconds
	if err := e.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		l.Log.Fatal(err)
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

	l.Log.Infof("Metrics Server started on :%v", port)
	l.Log.Fatal(e.Start(":" + port))
}
