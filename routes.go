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

var listMiddleware = []echo.MiddlewareFunc{
	middleware.SortAndFilter, middleware.Pagination,
}

var tenancyWithListMiddleware = append([]echo.MiddlewareFunc{enforceTenancy}, listMiddleware...)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

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
	v3.GET("/sources", SourceList, tenancyWithListMiddleware...)
	v3.GET("/sources/:id", SourceGet, enforceTenancy)
	v3.POST("/sources", SourceCreate, enforceTenancy)
	v3.PATCH("/sources/:id", SourceEdit, enforceTenancy)
	v3.DELETE("/sources/:id", SourceDelete, enforceTenancy)
	v3.GET("/sources/:source_id/application_types", SourceListApplicationTypes, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/applications", SourceListApplications, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/endpoints", SourceListEndpoint, tenancyWithListMiddleware...)

	// Applications
	v3.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
	v3.GET("/applications/:id", ApplicationGet, enforceTenancy)

	// Authentications
	v3.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
	v3.POST("/authentications", AuthenticationCreate, enforceTenancy)
	v3.GET("/authentications/:uid", AuthenticationGet, enforceTenancy)
	v3.PATCH("/authentications/:uid", AuthenticationUpdate, enforceTenancy)
	v3.DELETE("/authentications/:uid", AuthenticationDelete, enforceTenancy)

	// ApplicationTypes
	v3.GET("/application_types", ApplicationTypeList, listMiddleware...)
	v3.GET("/application_types/:id", ApplicationTypeGet)
	v3.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, tenancyWithListMiddleware...)

	// Endpoints
	v3.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
	v3.GET("/endpoints/:id", EndpointGet, enforceTenancy)

	// ApplicationAuthentications
	v3.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
	v3.GET("/application_authentications/:id", ApplicationAuthenticationGet, enforceTenancy)

	// AppMetaData
	v3.GET("/app_meta_data", MetaDataList, listMiddleware...)
	v3.GET("/app_meta_data/:id", MetaDataGet)
	v3.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, listMiddleware...)

	// SourceTypes
	v3.GET("/source_types", SourceTypeList, listMiddleware...)
	v3.GET("/source_types/:id", SourceTypeGet)
	v3.GET("/source_types/:source_type_id/sources", SourceTypeListSource, tenancyWithListMiddleware...)
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
