package main

import (
	"embed"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed public
var publicDocs embed.FS

func PublicOpenApi(version string) echo.HandlerFunc {
	return func(c echo.Context) error {
		bytes, err := publicDocs.ReadFile("public/openapi-3-" + version + ".json")
		if err != nil {
			return err
		}

		return c.JSONBlob(http.StatusOK, bytes)
	}
}
