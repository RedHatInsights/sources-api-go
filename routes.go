package main

import (
	"net/http"

	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/labstack/echo/v4"
)

var listMiddleware = []echo.MiddlewareFunc{
	middleware.SortAndFilter, middleware.Pagination,
}

var tenancyWithListMiddleware = append([]echo.MiddlewareFunc{middleware.Tenancy}, listMiddleware...)
var permissionMiddleware = []echo.MiddlewareFunc{middleware.Tenancy, middleware.PermissionCheck, middleware.RaiseEvent}
var permissionWithListMiddleware = append(listMiddleware, middleware.PermissionCheck)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	v3 := e.Group("/api/sources/v3.1", middleware.Timing, middleware.HandleErrors, middleware.ParseHeaders)

	//openapi
	v3.GET("/openapi.json", PublicOpenApiv31)

	// Sources
	v3.GET("/sources", SourceList, tenancyWithListMiddleware...)
	v3.GET("/sources/:id", SourceGet, middleware.Tenancy)
	v3.POST("/sources", SourceCreate, permissionMiddleware...)
	v3.PATCH("/sources/:id", SourceEdit, permissionMiddleware...)
	v3.DELETE("/sources/:id", SourceDelete, append(permissionMiddleware, middleware.SuperKeyDestroySource)...)
	v3.POST("/sources/:source_id/check_availability", SourceCheckAvailability, middleware.Tenancy)
	v3.GET("/sources/:source_id/application_types", SourceListApplicationTypes, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/applications", SourceListApplications, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/endpoints", SourceListEndpoint, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/authentications", SourceListAuthentications, tenancyWithListMiddleware...)
	v3.GET("/sources/:source_id/rhc_connections", SourcesRhcConnectionList, tenancyWithListMiddleware...)
	v3.POST("/sources/:source_id/pause", SourcePause, middleware.Tenancy)
	v3.POST("/sources/:source_id/unpause", SourceUnpause, middleware.Tenancy)

	// Applications
	v3.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
	v3.GET("/applications/:id", ApplicationGet, middleware.Tenancy)
	v3.POST("/applications", ApplicationCreate, permissionMiddleware...)
	v3.PATCH("/applications/:id", ApplicationEdit, permissionMiddleware...)
	v3.DELETE("/applications/:id", ApplicationDelete, append(permissionMiddleware, middleware.SuperKeyDestroyApplication)...)
	v3.GET("/applications/:application_id/authentications", ApplicationListAuthentications, tenancyWithListMiddleware...)
	v3.POST("/applications/:id/pause", ApplicationPause, middleware.Tenancy)
	v3.POST("/applications/:id/unpause", ApplicationUnpause, middleware.Tenancy)

	// Authentications
	v3.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
	v3.GET("/authentications/:uid", AuthenticationGet, middleware.Tenancy)
	v3.POST("/authentications", AuthenticationCreate, permissionMiddleware...)
	v3.PATCH("/authentications/:uid", AuthenticationUpdate, permissionMiddleware...)
	v3.DELETE("/authentications/:uid", AuthenticationDelete, permissionMiddleware...)

	// ApplicationTypes
	v3.GET("/application_types", ApplicationTypeList, listMiddleware...)
	v3.GET("/application_types/:id", ApplicationTypeGet)
	v3.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, tenancyWithListMiddleware...)

	// Endpoints
	v3.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
	v3.GET("/endpoints/:id", EndpointGet, middleware.Tenancy)
	v3.POST("/endpoints", EndpointCreate, permissionMiddleware...)
	v3.DELETE("/endpoints/:id", EndpointDelete, permissionMiddleware...)
	v3.GET("/endpoints/:endpoint_id/authentications", EndpointListAuthentications, tenancyWithListMiddleware...)

	// ApplicationAuthentications
	v3.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
	v3.GET("/application_authentications/:id", ApplicationAuthenticationGet, middleware.Tenancy)
	v3.GET("/application_authentications/:application_authentication_id/authentications", ApplicationAuthenticationListAuthentications, tenancyWithListMiddleware...)
	v3.POST("/application_authentications", ApplicationAuthenticationCreate, permissionMiddleware...)
	v3.DELETE("/application_authentications/:id", ApplicationAuthenticationDelete, permissionMiddleware...)

	// AppMetaData
	v3.GET("/app_meta_data", MetaDataList, listMiddleware...)
	v3.GET("/app_meta_data/:id", MetaDataGet)
	v3.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, listMiddleware...)

	// SourceTypes
	v3.GET("/source_types", SourceTypeList, listMiddleware...)
	v3.GET("/source_types/:id", SourceTypeGet)
	v3.GET("/source_types/:source_type_id/sources", SourceTypeListSource, tenancyWithListMiddleware...)

	// Red Hat Connector Connections
	v3.GET("/rhc_connections", RhcConnectionList, tenancyWithListMiddleware...)
	v3.GET("/rhc_connections/:id", RhcConnectionGetById, permissionMiddleware...)
	v3.POST("/rhc_connections", RhcConnectionCreate, permissionMiddleware...)
	v3.PATCH("/rhc_connections/:id", RhcConnectionUpdate, permissionMiddleware...)
	v3.DELETE("/rhc_connections/:id", RhcConnectionDelete, permissionMiddleware...)
	v3.GET("/rhc_connections/:id/sources", RhcConnectionSourcesList, permissionWithListMiddleware...)

	// GraphQL
	v3.POST("/graphql", ProxyGraphqlToLegacySources)

	/**            **\
	 * Internal API *
	\**            **/
	internal := e.Group("/internal/v2.0", middleware.HandleErrors, middleware.ParseHeaders)

	// Authentications
	internal.GET("/authentications/:uuid", InternalAuthenticationGet, permissionMiddleware...)

	// Sources
	internal.GET("/sources", InternalSourceList, permissionWithListMiddleware...)
}
