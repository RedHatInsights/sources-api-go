package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func ApplicationList(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func ApplicationGet(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func ApplicationCreate(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func ApplicationEdit(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func ApplicationDelete(c echo.Context) (err error) {
	return c.String(http.StatusOK, "hello")
}
