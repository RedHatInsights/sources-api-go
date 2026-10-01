package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RedHatInsights/sources-api-go/mcp/config"
	"github.com/RedHatInsights/sources-api-go/mcp/server"
	echoprometheus "github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	cfg := config.Get()

	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{})

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(echoprometheus.NewMiddleware("sources_mcp"))

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	mcpServer, err := server.NewMCPServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	e.POST("/_private/mcp", mcpServer.HandleHTTP)

	go runMetricsServer(cfg.MetricsPort)

	go func() {
		log.Infof("MCP Server starting on :%d", cfg.Port)
		if err := e.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Info("Server shutdown complete")
}

func runMetricsServer(port int) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/metrics", echoprometheus.NewHandler())

	log.Infof("Metrics server starting on :%d", port)
	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
