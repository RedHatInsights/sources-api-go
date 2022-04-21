package main

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
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

	apiVersions := []string{"v1.0", "v2.0", "v3.0", "v3.1"}
	for _, version := range apiVersions {
		r := e.Group("/api/sources/"+version, middleware.Timing, middleware.HandleErrors, middleware.ParseHeaders)

		// openapi
		r.GET("/openapi.json", PublicOpenApi(version))

		// Bulk Create
		r.POST("/bulk_create", BulkCreate, permissionMiddleware...)

		// Sources
		r.GET("/sources", SourceList, tenancyWithListMiddleware...)
		r.GET("/sources/:id", SourceGet, middleware.Tenancy)
		r.POST("/sources", SourceCreate, permissionMiddleware...)
		r.PATCH("/sources/:id", SourceEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/sources/:id", SourceDelete, append(permissionMiddleware, middleware.SuperKeyDestroySource)...)
		r.POST("/sources/:source_id/check_availability", SourceCheckAvailability, middleware.Tenancy)
		r.GET("/sources/:source_id/application_types", SourceListApplicationTypes, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/applications", SourceListApplications, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/endpoints", SourceListEndpoint, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/authentications", SourceListAuthentications, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/rhc_connections", SourcesRhcConnectionList, tenancyWithListMiddleware...)
		r.POST("/sources/:source_id/pause", SourcePause, middleware.Tenancy)
		r.POST("/sources/:source_id/unpause", SourceUnpause, middleware.Tenancy)

		// Applications
		r.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
		r.GET("/applications/:id", ApplicationGet, middleware.Tenancy)
		r.POST("/applications", ApplicationCreate, permissionMiddleware...)
		r.PATCH("/applications/:id", ApplicationEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/applications/:id", ApplicationDelete, append(permissionMiddleware, middleware.SuperKeyDestroyApplication)...)
		r.GET("/applications/:application_id/authentications", ApplicationListAuthentications, tenancyWithListMiddleware...)
		r.POST("/applications/:id/pause", ApplicationPause, middleware.Tenancy)
		r.POST("/applications/:id/unpause", ApplicationUnpause, middleware.Tenancy)

		// Authentications
		r.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
		r.GET("/authentications/:uid", AuthenticationGet, middleware.Tenancy)
		r.POST("/authentications", AuthenticationCreate, permissionMiddleware...)
		r.PATCH("/authentications/:uid", AuthenticationEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/authentications/:uid", AuthenticationDelete, permissionMiddleware...)

		// ApplicationTypes
		r.GET("/application_types", ApplicationTypeList, listMiddleware...)
		r.GET("/application_types/:id", ApplicationTypeGet)
		r.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, tenancyWithListMiddleware...)

		// Endpoints
		r.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
		r.GET("/endpoints/:id", EndpointGet, middleware.Tenancy)
		r.POST("/endpoints", EndpointCreate, permissionMiddleware...)
		r.PATCH("/endpoints/:id", EndpointEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/endpoints/:id", EndpointDelete, permissionMiddleware...)
		r.GET("/endpoints/:endpoint_id/authentications", EndpointListAuthentications, tenancyWithListMiddleware...)

		// ApplicationAuthentications
		r.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
		r.GET("/application_authentications/:id", ApplicationAuthenticationGet, middleware.Tenancy)
		r.GET("/application_authentications/:application_authentication_id/authentications", ApplicationAuthenticationListAuthentications, tenancyWithListMiddleware...)
		r.POST("/application_authentications", ApplicationAuthenticationCreate, permissionMiddleware...)
		r.DELETE("/application_authentications/:id", ApplicationAuthenticationDelete, permissionMiddleware...)

		// AppMetaData
		r.GET("/app_meta_data", MetaDataList, listMiddleware...)
		r.GET("/app_meta_data/:id", MetaDataGet)
		r.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, listMiddleware...)

		// SourceTypes
		r.GET("/source_types", SourceTypeList, listMiddleware...)
		r.GET("/source_types/:id", SourceTypeGet)
		r.GET("/source_types/:source_type_id/sources", SourceTypeListSource, tenancyWithListMiddleware...)

		// Red Hat Connector Connections
		r.GET("/rhc_connections", RhcConnectionList, tenancyWithListMiddleware...)
		r.GET("/rhc_connections/:id", RhcConnectionGetById, permissionMiddleware...)
		r.POST("/rhc_connections", RhcConnectionCreate, permissionMiddleware...)
		r.PATCH("/rhc_connections/:id", RhcConnectionEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/rhc_connections/:id", RhcConnectionDelete, permissionMiddleware...)
		r.GET("/rhc_connections/:id/sources", RhcConnectionSourcesList, permissionWithListMiddleware...)

		// GraphQL
		// TODO: remove this once we get the crazy filtering going on the gqlgen graphql
		if os.Getenv("PROXY_GRAPHQL") == "true" {
			r.POST("/graphql", ProxyGraphqlToLegacySources, middleware.Tenancy)
		} else {
			r.POST("/graphql", GraphQLQuery, middleware.Tenancy)

			// run the graphQL playground if running locally or in ephemeral. really handy for development!
			// https://github.com/graphql/graphiql
			if os.Getenv("SOURCES_ENV") != "stage" && os.Getenv("SOURCES_ENV") != "prod" {
				r.GET("/graphql_playground", echo.WrapHandler(playground.Handler("Sources API GraphQL Playground", "/api/sources/v3.1/graphql")))
			}
		}
	}

	/**            **\
	 * Internal API *
	\**            **/
	internalv2 := e.Group("/internal/v2.0", middleware.HandleErrors, middleware.ParseHeaders)

	// Authentications
	internalv2.GET("/authentications/:uuid", InternalAuthenticationGet, permissionMiddleware...)

	// Sources
	internalv2.GET("/sources", InternalSourceList, permissionWithListMiddleware...)

	/**            **\
	 * Internal API *
	\**            **/
	internvalv1 := e.Group("/internal/v1.0", middleware.HandleErrors, middleware.ParseHeaders)

	// Authentications
	internvalv1.GET("/authentications/:uuid", InternalAuthenticationGet, permissionMiddleware...)

	// Sources
	internvalv1.GET("/sources", InternalSourceList, permissionWithListMiddleware...)
}
