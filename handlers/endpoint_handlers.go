package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func EndpointList(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func EndpointGet(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func EndpointCreate(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func EndpointEdit(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func EndpointDelete(c echo.Context) (err error) {
	return c.String(http.StatusOK, "hello")
}
