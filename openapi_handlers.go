package main

import (
	"embed"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed public
var publicDocs embed.FS

func PublicOpenApiv31(c echo.Context) error {
	bytes, err := readOpenApiJson("public/openapi-3-v3.1.json")
	if err != nil {
		return err
	}

	return c.JSONBlob(http.StatusOK, bytes)
}

func readOpenApiJson(filename string) ([]byte, error) {
	return publicDocs.ReadFile(filename)
}
