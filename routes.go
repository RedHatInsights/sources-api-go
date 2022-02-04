package main

import (
	"io/ioutil"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/labstack/echo/v4"
)

var listMiddleware = []echo.MiddlewareFunc{
	middleware.SortAndFilter, middleware.Pagination,
}

var tenancyWithListMiddleware = append([]echo.MiddlewareFunc{middleware.Tenancy}, listMiddleware...)
var permissionMiddleware = []echo.MiddlewareFunc{middleware.Tenancy, middleware.PermissionCheck}
var permissionWithListMiddleware = append(listMiddleware, middleware.PermissionCheck)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// TODO: pull this out into its own handler.
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

	v3 := e.Group("/api/sources/v3.1", middleware.HandleErrors, middleware.ParseHeaders)

	// Sources
	v3.GET("/sources", SourceList, tenancyWithListMiddleware...)
	v3.GET("/sources/:id", SourceGet, middleware.Tenancy)
	v3.POST("/sources", SourceCreate, append(permissionMiddleware, middleware.RaiseSourceCreateEvent)...)
	v3.PATCH("/sources/:id", SourceEdit, append(permissionMiddleware, middleware.RaiseSourceUpdateEvent)...)
	v3.DELETE("/sources/:id", SourceDelete, append(permissionMiddleware, middleware.RaiseSourceDestroyEvent)...)
	v3.POST("/sources/:source_id/check_availability", SourceCheckAvailability, middleware.Tenancy)
	v3.GET("/sources/:source_id/application_types", SourceListApplicationTypes, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/applications", SourceListApplications, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/endpoints", SourceListEndpoint, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/authentications", SourceListAuthentications, tenancyWithListMiddleware...)

	// Applications
	v3.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
	v3.GET("/applications/:id", ApplicationGet, middleware.Tenancy)
	v3.POST("/applications", ApplicationCreate, append(permissionMiddleware, middleware.RaiseApplicationCreateEvent)...)
	v3.PATCH("/applications/:id", ApplicationEdit, append(permissionMiddleware, middleware.RaiseApplicationUpdateEvent)...)
	v3.DELETE("/applications/:id", ApplicationDelete, append(permissionMiddleware, middleware.RaiseApplicationDestroyEvent)...)
	v3.GET("/applications/:application_id/authentications", ApplicationListAuthentications, tenancyWithListMiddleware...)

	// Authentications
	v3.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
	v3.GET("/authentications/:uid", AuthenticationGet, middleware.Tenancy)
	v3.POST("/authentications", AuthenticationCreate, append(permissionMiddleware, middleware.RaiseAuthenticationCreateEvent)...)
	v3.PATCH("/authentications/:uid", AuthenticationUpdate, append(permissionMiddleware, middleware.RaiseAuthenticationUpdateEvent)...)
	v3.DELETE("/authentications/:uid", AuthenticationDelete, append(permissionMiddleware, middleware.RaiseAuthenticationDestroyEvent)...)

	// ApplicationTypes
	v3.GET("/application_types", ApplicationTypeList, listMiddleware...)
	v3.GET("/application_types/:id", ApplicationTypeGet)
	v3.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, tenancyWithListMiddleware...)

	// Endpoints
	v3.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
	v3.GET("/endpoints/:id", EndpointGet, middleware.Tenancy)
	v3.POST("/endpoints", EndpointCreate, append(permissionMiddleware, middleware.RaiseEndpointCreateEvent)...)
	v3.DELETE("/endpoints/:id", EndpointDelete, append(permissionMiddleware, middleware.RaiseEndpointDestroyEvent)...)
	v3.GET("/endpoints/:endpoint_id/authentications", EndpointListAuthentications, tenancyWithListMiddleware...)

	// ApplicationAuthentications
	v3.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
	v3.GET("/application_authentications/:id", ApplicationAuthenticationGet, middleware.Tenancy)
	v3.GET("/application_authentications/:application_authentication_id/authentications", ApplicationAuthenticationListAuthentications, tenancyWithListMiddleware...)

	// AppMetaData
	v3.GET("/app_meta_data", MetaDataList, listMiddleware...)
	v3.GET("/app_meta_data/:id", MetaDataGet)
	v3.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, listMiddleware...)

	// SourceTypes
	v3.GET("/source_types", SourceTypeList, listMiddleware...)
	v3.GET("/source_types/:id", SourceTypeGet)
	v3.GET("/source_types/:source_type_id/sources", SourceTypeListSource, tenancyWithListMiddleware...)

	/**            **\
	 * Internal API *
	\**            **/
	internal := e.Group("/internal/v2.0", middleware.HandleErrors, middleware.ParseHeaders)

	internal.GET("/sources", InternalSourceList, permissionWithListMiddleware...)
}
