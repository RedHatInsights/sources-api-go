package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/lindgrenj6/sources-api-go/dao"
	"github.com/lindgrenj6/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/handlers"
	. "github.com/lindgrenj6/sources-api-go/middleware"
	"github.com/lindgrenj6/sources-api-go/redis"
)

func setupRoutes(e *echo.Echo) {
	e.GET("/openapi.json", func(c echo.Context) error {
		out, err := redis.Client.Get("openapi").Result()
		if err != nil {
			file, err := ioutil.ReadFile("openapi-3-v3.1.json")
			if err != nil {
				return c.NoContent(http.StatusBadRequest)
			}
			err = redis.Client.Set("openapi", string(file), -1).Err()
			if err != nil {
				return c.NoContent(http.StatusBadRequest)
			}
			out = string(file)
		}
		return c.String(http.StatusOK, out)
	})

	v3 := e.Group("/api/sources/v3.1", enforceTenancy)

	v3.GET("/sources", handlers.SourceList, ParseFilter, ParsePagination)
	v3.GET("/sources/:id", handlers.SourceGet)
	v3.POST("/sources", handlers.SourceCreate)
	v3.PATCH("/sources/:id", handlers.SourceEdit)
	v3.DELETE("/sources/:id", handlers.SourceDelete)

	v3.GET("/endpoints", handlers.EndpointList)
	v3.GET("/endpoints/:id", handlers.EndpointGet)
	v3.POST("/endpoints", handlers.EndpointCreate)
	v3.PATCH("/endpoints/:id", handlers.EndpointEdit)
	v3.DELETE("/endpoints/:id", handlers.EndpointDelete)

	v3.GET("/applications", handlers.ApplicationList)
	v3.GET("/applications/:id", handlers.ApplicationGet)
	v3.POST("/applications", handlers.ApplicationCreate)
	v3.PATCH("/applications/:id", handlers.ApplicationEdit)
	v3.DELETE("/applications/:id", handlers.ApplicationDelete)

	v3.GET("/authentications", handlers.AuthenticationList)
	v3.GET("/authentications/:id", handlers.AuthenticationGet)
	v3.POST("/authentications", handlers.AuthenticationCreate)
	v3.PATCH("/authentications/:id", handlers.AuthenticationEdit)
	v3.DELETE("/authentications/:id", handlers.AuthenticationDelete)
}

func enforceTenancy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case c.Request().Header.Get("x-rh-sources-account-number") != "":
			accountNumber := c.Request().Header.Get("x-rh-sources-account-number")
			t := dao.GetOrCreateTenantID(accountNumber)
			c.Set("tenantID", *t)

		case c.Request().Header.Get("x-rh-identity") != "":
			idRaw, err := base64.StdEncoding.DecodeString(c.Request().Header.Get("x-rh-identity"))
			if err != nil {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc(fmt.Sprintf("Error decoding Identity: %v", err), "401"))
			}

			var jsonData identity.XRHID
			err = json.Unmarshal(idRaw, &jsonData)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("x-rh-identity header is does not contain valid JSON", "401"))
			}

			if jsonData.Identity.AccountNumber == "" {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("No Tenant Id!", "401"))
			}

			t := dao.GetOrCreateTenantID(jsonData.Identity.AccountNumber)
			c.Set("tenantID", *t)

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("No Tenant Id!", "401"))
		}
		return next(c)
	}
}
