package main

import (
	"embed"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed public
var publicDocs embed.FS

func PublicOpenApiv31(c echo.Context) error {
	out, err := readOpenApiJson("public/openapi-3-v3.1.json")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, out)
}

func readOpenApiJson(filename string) (map[string]interface{}, error) {
	bytes, err := publicDocs.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	err = json.Unmarshal(bytes, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
