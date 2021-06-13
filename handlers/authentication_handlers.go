package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func AuthenticationList(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func AuthenticationGet(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func AuthenticationCreate(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func AuthenticationEdit(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func AuthenticationDelete(c echo.Context) (err error) {
	return c.String(http.StatusOK, "hello")
}
