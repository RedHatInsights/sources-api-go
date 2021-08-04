package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func setupRoutes(e *echo.Echo) {
	e.GET("/openapi.json", func(c echo.Context) error {
		out, err := redis.Client.Get("openapi").Result()
		if err != nil {
			file, err := ioutil.ReadFile("public/openapi-3-v3.1.json")
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

	v3 := e.Group("/api/sources/v3.1", middleware.HandleErrors)

	// Sources
	v3.GET("/sources", SourceList, enforceTenancy, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/sources/:id", SourceGet, enforceTenancy)
	v3.POST("/sources", SourceCreate, enforceTenancy)
	v3.PATCH("/sources/:id", SourceEdit, enforceTenancy)
	v3.DELETE("/sources/:id", SourceDelete, enforceTenancy)
	v3.GET("/sources/:source_id/application_types", SourceListApplicationTypes, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/sources/:source_id/applications", SourceListApplications, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/sources/:source_id/endpoints", SourceListEndpoint, middleware.SortAndFilter, middleware.Pagination)

	// Applications
	v3.GET("/applications", ApplicationList, enforceTenancy, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/applications/:id", ApplicationGet, enforceTenancy)

	// ApplicationTypes
	v3.GET("/application_types", ApplicationTypeList, middleware.SortAndFilter, middleware.Pagination, withoutTenancy)
	v3.GET("/application_types/:id", ApplicationTypeGet, withoutTenancy)
	v3.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, middleware.SortAndFilter, middleware.Pagination)

	// Endpoints
	v3.GET("/endpoints", EndpointList, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/endpoints/:id", EndpointGet)

	// ApplicationAuthentications
	v3.GET("/application_authentications", ApplicationAuthenticationList, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/application_authentications/:id", ApplicationAuthenticationGet)

	v3.GET("/app_meta_data", MetaDataList, middleware.SortAndFilter, middleware.Pagination, withoutTenancy)
	v3.GET("/app_meta_data/:id", MetaDataGet, withoutTenancy)
	v3.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, middleware.SortAndFilter, middleware.Pagination)

	// SourceTypes
	v3.GET("/source_types", SourceTypeList, middleware.SortAndFilter, middleware.Pagination)
	v3.GET("/source_types/:id", SourceTypeGet)
	v3.GET("/source_types/:source_type_id/sources", SourceTypeListSource, middleware.SortAndFilter, middleware.Pagination)
}

func withoutTenancy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("withoutTenancy", true)
		return next(c)
	}
}

func enforceTenancy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case c.Request().Header.Get("x-rh-sources-account-number") != "":
			accountNumber := c.Request().Header.Get("x-rh-sources-account-number")
			c.Logger().Debugf("Looking up Tenant ID for account number %v", accountNumber)
			t, err := dao.GetOrCreateTenantID(accountNumber)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, util.ErrorDoc("Failed to get or create tenant for request", "500"))
			}
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

			c.Logger().Debugf("Looking up Tenant ID for account number %v", jsonData.Identity.AccountNumber)
			t, err := dao.GetOrCreateTenantID(jsonData.Identity.AccountNumber)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, util.ErrorDoc("Failed to get or create tenant for request", "500"))
			}
			c.Set("tenantID", *t)

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("No Tenant Id!", "401"))
		}
		return next(c)
	}
}
