package main

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/middleware"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var listMiddleware = []echo.MiddlewareFunc{
	middleware.SortAndFilter, middleware.Pagination,
}

var tenancyWithListMiddleware = append([]echo.MiddlewareFunc{middleware.Tenancy, middleware.UserCatcher}, listMiddleware...)
var permissionMiddleware = []echo.MiddlewareFunc{middleware.Tenancy, middleware.UserCatcher, middleware.PermissionCheck, middleware.RaiseEvent}
var permissionWithListMiddleware = append(listMiddleware, middleware.PermissionCheck)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// overriding the echo.Context instance with our own - so we can use any
	// changes we made to the context's methods.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { return next(&util.SourcesContext{Context: c}) }
	})

	apiVersions := []string{"v1.0", "v2.0", "v3.0", "v3.1", "v1", "v2", "v3"}
	for _, version := range apiVersions {
		r := e.Group("/api/sources/"+version, middleware.Timing, middleware.HandleErrors, middleware.ParseHeaders, middleware.LoggerFields)

		// openapi
		r.GET("/openapi.json", PublicOpenApi(version))

		// Bulk Create
		r.POST("/bulk_create", BulkCreate, permissionMiddleware...)

		// Sources
		r.GET("/sources", SourceList, tenancyWithListMiddleware...)
		r.GET("/sources/:id", SourceGet, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
		r.POST("/sources", SourceCreate, permissionMiddleware...)
		r.PATCH("/sources/:id", SourceEdit, append(permissionMiddleware, middleware.Notifier, middleware.IdValidation)...)
		r.DELETE("/sources/:id", SourceDelete, append(permissionMiddleware, middleware.SuperKeyDestroySource, middleware.IdValidation)...)
		r.POST("/sources/:source_id/check_availability", SourceCheckAvailability, middleware.Tenancy, middleware.IdValidation)
		r.GET("/sources/:source_id/application_types", SourceListApplicationTypes, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.GET("/sources/:source_id/applications", SourceListApplications, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.GET("/sources/:source_id/endpoints", SourceListEndpoint, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.GET("/sources/:source_id/authentications", SourceListAuthentications, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.GET("/sources/:source_id/rhc_connections", SourcesRhcConnectionList, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.POST("/sources/:source_id/pause", SourcePause, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
		r.POST("/sources/:source_id/unpause", SourceUnpause, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)

		// Applications
		r.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
		r.GET("/applications/:id", ApplicationGet, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
		r.POST("/applications", ApplicationCreate, permissionMiddleware...)
		r.PATCH("/applications/:id", ApplicationEdit, append(permissionMiddleware, middleware.Notifier, middleware.IdValidation)...)
		r.DELETE("/applications/:id", ApplicationDelete, append(permissionMiddleware, middleware.SuperKeyDestroyApplication, middleware.IdValidation)...)
		r.GET("/applications/:application_id/authentications", ApplicationListAuthentications, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.POST("/applications/:id/pause", ApplicationPause, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
		r.POST("/applications/:id/unpause", ApplicationUnpause, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)

		// Authentications
		r.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
		r.POST("/authentications", AuthenticationCreate, permissionMiddleware...)
		if config.IsVaultOn() {
			r.GET("/authentications/:uid", AuthenticationGet, middleware.Tenancy, middleware.UserCatcher, middleware.UuidValidation)
			r.PATCH("/authentications/:uid", AuthenticationEdit, append(permissionMiddleware, middleware.Notifier, middleware.UuidValidation)...)
			r.DELETE("/authentications/:uid", AuthenticationDelete, append(permissionMiddleware, middleware.UuidValidation)...)
		} else {
			r.GET("/authentications/:uid", AuthenticationGet, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
			r.PATCH("/authentications/:uid", AuthenticationEdit, append(permissionMiddleware, middleware.Notifier, middleware.IdValidation)...)
			r.DELETE("/authentications/:uid", AuthenticationDelete, append(permissionMiddleware, middleware.IdValidation)...)
		}

		// ApplicationTypes
		r.GET("/application_types", ApplicationTypeList, listMiddleware...)
		r.GET("/application_types/:id", ApplicationTypeGet, middleware.IdValidation)
		r.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, append(tenancyWithListMiddleware, middleware.IdValidation)...)

		// Endpoints
		r.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
		r.GET("/endpoints/:id", EndpointGet, middleware.Tenancy, middleware.IdValidation)
		r.POST("/endpoints", EndpointCreate, permissionMiddleware...)
		r.PATCH("/endpoints/:id", EndpointEdit, append(permissionMiddleware, middleware.Notifier, middleware.IdValidation)...)
		r.DELETE("/endpoints/:id", EndpointDelete, append(permissionMiddleware, middleware.IdValidation)...)
		r.GET("/endpoints/:endpoint_id/authentications", EndpointListAuthentications, append(tenancyWithListMiddleware, middleware.IdValidation)...)

		// ApplicationAuthentications
		r.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
		r.GET("/application_authentications/:id", ApplicationAuthenticationGet, middleware.Tenancy, middleware.UserCatcher, middleware.IdValidation)
		r.GET("/application_authentications/:application_authentication_id/authentications", ApplicationAuthenticationListAuthentications, append(tenancyWithListMiddleware, middleware.IdValidation)...)
		r.POST("/application_authentications", ApplicationAuthenticationCreate, permissionMiddleware...)
		r.DELETE("/application_authentications/:id", ApplicationAuthenticationDelete, append(permissionMiddleware, middleware.IdValidation)...)

		// AppMetaData
		r.GET("/app_meta_data", MetaDataList, listMiddleware...)
		r.GET("/app_meta_data/:id", MetaDataGet, middleware.IdValidation)
		r.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, append(listMiddleware, middleware.IdValidation)...)

		// SourceTypes
		r.GET("/source_types", SourceTypeList, listMiddleware...)
		r.GET("/source_types/:id", SourceTypeGet, middleware.IdValidation)
		r.GET("/source_types/:source_type_id/sources", SourceTypeListSource, append(tenancyWithListMiddleware, middleware.IdValidation)...)

		// Red Hat Connector Connections
		r.GET("/rhc_connections", RhcConnectionList, tenancyWithListMiddleware...)
		r.GET("/rhc_connections/:id", RhcConnectionGetById, append(permissionMiddleware, middleware.IdValidation)...)
		r.POST("/rhc_connections", RhcConnectionCreate, permissionMiddleware...)
		r.PATCH("/rhc_connections/:id", RhcConnectionEdit, append(permissionMiddleware, middleware.Notifier, middleware.IdValidation)...)
		r.DELETE("/rhc_connections/:id", RhcConnectionDelete, append(permissionMiddleware, middleware.IdValidation)...)
		r.GET("/rhc_connections/:id/sources", RhcConnectionSourcesList, append(permissionWithListMiddleware, middleware.IdValidation)...)

		// GraphQL
		r.POST("/graphql", GraphQLQuery, middleware.Tenancy, middleware.UserCatcher)

		// run the graphQL playground if running locally or in ephemeral. really handy for development!
		// https://github.com/graphql/graphiql
		if os.Getenv("SOURCES_ENV") != "stage" && os.Getenv("SOURCES_ENV") != "prod" {
			r.GET("/graphql_playground", echo.WrapHandler(playground.Handler("Sources API GraphQL Playground", "/api/sources/v3.1/graphql")))
		}
	}

	/**            **\
	 * Internal API *
	\**            **/
	internalVersions := []string{"v1.0", "v2.0"}
	for _, version := range internalVersions {
		r := e.Group("/internal/"+version, middleware.HandleErrors, middleware.ParseHeaders, middleware.LoggerFields)

		// Authentications
		r.GET("/authentications/:uuid", InternalAuthenticationGet, permissionMiddleware...)
		// Sources
		r.GET("/sources", InternalSourceList, permissionWithListMiddleware...)
	}
}
