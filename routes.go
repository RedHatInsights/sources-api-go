package main

import (
	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/handlers"
)

func setupRoutes(e *echo.Echo) {
	e.GET("/sources", handlers.SourceList)
	e.GET("/sources/:id", handlers.SourceGet)
	e.POST("/sources", handlers.SourceCreate)
	e.PATCH("/sources/:id", handlers.SourceEdit)
	e.DELETE("/sources/:id", handlers.SourceDelete)

	e.GET("/endpoints", handlers.EndpointList)
	e.GET("/endpoints/:id", handlers.EndpointGet)
	e.POST("/endpoints", handlers.EndpointCreate)
	e.PATCH("/endpoints/:id", handlers.EndpointEdit)
	e.DELETE("/endpoints/:id", handlers.EndpointDelete)

	e.GET("/applications", handlers.ApplicationList)
	e.GET("/applications/:id", handlers.ApplicationGet)
	e.POST("/applications", handlers.ApplicationCreate)
	e.PATCH("/applications/:id", handlers.ApplicationEdit)
	e.DELETE("/applications/:id", handlers.ApplicationDelete)

	e.GET("/authentications", handlers.AuthenticationList)
	e.GET("/authentications/:id", handlers.AuthenticationGet)
	e.POST("/authentications", handlers.AuthenticationCreate)
	e.PATCH("/authentications/:id", handlers.AuthenticationEdit)
	e.DELETE("/authentications/:id", handlers.AuthenticationDelete)
}
