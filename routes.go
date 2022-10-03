package main

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/middleware"
	echoUtils "github.com/RedHatInsights/sources-api-go/util/echo"
	"github.com/labstack/echo/v4"
)

var listMiddleware = []echo.MiddlewareFunc{middleware.SortAndFilter, middleware.Pagination}
var tenancyMiddleware = []echo.MiddlewareFunc{middleware.Tenancy, middleware.LoggerFields, middleware.UserCatcher}

var tenancyWithListMiddleware = append(tenancyMiddleware, listMiddleware...)
var permissionMiddleware = append(permissionMiddlewareWithoutEvents, middleware.RaiseEvent)
var permissionWithListMiddleware = append(listMiddleware, middleware.PermissionCheck)
var permissionMiddlewareWithoutEvents = append(tenancyMiddleware, middleware.PermissionCheck)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// overriding the echo.Context instance with our own - so we can use any
	// changes we made to the context's methods.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { return next(&echoUtils.SourcesContext{Context: c}) }
	})

	apiVersions := []string{"v1.0", "v2.0", "v3.0", "v3.1", "v1", "v2", "v3"}
	for _, version := range apiVersions {
		// this is the "base" middleware set, used on every call
		r := e.Group("/api/sources/"+version,
			middleware.Timing,
			middleware.HandleErrors,
			middleware.IdValidation,
			middleware.ParseHeaders,
		)

		// openapi
		r.GET("/openapi.json", PublicOpenApi(version))

		// Bulk Create
		r.POST("/bulk_create", BulkCreate, permissionMiddleware...)

		// Sources
		r.GET("/sources", SourceList, tenancyWithListMiddleware...)
		r.GET("/sources/:id", SourceGet, tenancyMiddleware...)
		r.POST("/sources", SourceCreate, permissionMiddleware...)
		r.PATCH("/sources/:id", SourceEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/sources/:id", SourceDelete, append(permissionMiddleware, middleware.SuperKeyDestroySource)...)
		r.POST("/sources/:source_id/check_availability", SourceCheckAvailability, middleware.Tenancy, middleware.LoggerFields)
		r.GET("/sources/:source_id/application_types", SourceListApplicationTypes, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/applications", SourceListApplications, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/endpoints", SourceListEndpoint, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/authentications", SourceListAuthentications, tenancyWithListMiddleware...)
		r.GET("/sources/:source_id/rhc_connections", SourcesRhcConnectionList, tenancyWithListMiddleware...)
		r.POST("/sources/:source_id/pause", SourcePause, tenancyMiddleware...)
		r.POST("/sources/:source_id/unpause", SourceUnpause, tenancyMiddleware...)

		// Applications
		r.GET("/applications", ApplicationList, tenancyWithListMiddleware...)
		r.GET("/applications/:id", ApplicationGet, tenancyMiddleware...)
		r.POST("/applications", ApplicationCreate, permissionMiddleware...)
		r.PATCH("/applications/:id", ApplicationEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/applications/:id", ApplicationDelete, append(permissionMiddleware, middleware.SuperKeyDestroyApplication)...)
		r.GET("/applications/:application_id/authentications", ApplicationListAuthentications, tenancyWithListMiddleware...)
		r.POST("/applications/:id/pause", ApplicationPause, tenancyMiddleware...)
		r.POST("/applications/:id/unpause", ApplicationUnpause, tenancyMiddleware...)

		// Authentications
		r.GET("/authentications", AuthenticationList, tenancyWithListMiddleware...)
		r.POST("/authentications", AuthenticationCreate, permissionMiddleware...)
		if config.IsVaultOn() {
			r.GET("/authentications/:uid", AuthenticationGet, append(tenancyMiddleware, middleware.UuidValidation)...)
			r.PATCH("/authentications/:uid", AuthenticationEdit, append(permissionMiddleware, middleware.Notifier, middleware.UuidValidation)...)
			r.DELETE("/authentications/:uid", AuthenticationDelete, append(permissionMiddleware, middleware.UuidValidation)...)
		} else {
			r.GET("/authentications/:uid", AuthenticationGet, tenancyMiddleware...)
			r.PATCH("/authentications/:uid", AuthenticationEdit, append(permissionMiddleware, middleware.Notifier)...)
			r.DELETE("/authentications/:uid", AuthenticationDelete, permissionMiddleware...)
		}

		// ApplicationTypes
		r.GET("/application_types", ApplicationTypeList, append(listMiddleware, middleware.LoggerFields)...)
		r.GET("/application_types/:id", ApplicationTypeGet, middleware.LoggerFields)
		r.GET("/application_types/:application_type_id/sources", ApplicationTypeListSource, tenancyWithListMiddleware...)

		// Endpoints
		r.GET("/endpoints", EndpointList, tenancyWithListMiddleware...)
		r.GET("/endpoints/:id", EndpointGet, middleware.Tenancy, middleware.LoggerFields)
		r.POST("/endpoints", EndpointCreate, permissionMiddleware...)
		r.PATCH("/endpoints/:id", EndpointEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/endpoints/:id", EndpointDelete, permissionMiddleware...)
		r.GET("/endpoints/:endpoint_id/authentications", EndpointListAuthentications, tenancyWithListMiddleware...)

		// ApplicationAuthentications
		r.GET("/application_authentications", ApplicationAuthenticationList, tenancyWithListMiddleware...)
		r.GET("/application_authentications/:id", ApplicationAuthenticationGet, tenancyMiddleware...)
		r.GET("/application_authentications/:application_authentication_id/authentications", ApplicationAuthenticationListAuthentications, tenancyWithListMiddleware...)
		r.POST("/application_authentications", ApplicationAuthenticationCreate, permissionMiddleware...)
		r.DELETE("/application_authentications/:id", ApplicationAuthenticationDelete, permissionMiddleware...)

		// AppMetaData
		r.GET("/app_meta_data", MetaDataList, append(listMiddleware, middleware.LoggerFields)...)
		r.GET("/app_meta_data/:id", MetaDataGet, middleware.LoggerFields)
		r.GET("/application_types/:application_type_id/app_meta_data", ApplicationTypeListMetaData, append(listMiddleware, middleware.LoggerFields)...)

		// Secrets
		r.GET("/secrets", SecretList, tenancyWithListMiddleware...)
		r.GET("/secrets/:id", SecretGet, tenancyMiddleware...)
		r.POST("/secrets", SecretCreate, permissionMiddlewareWithoutEvents...)
		r.PATCH("/secrets/:id", SecretEdit, permissionMiddlewareWithoutEvents...)
		r.DELETE("/secrets/:id", SecretDelete, permissionMiddleware...)

		// SourceTypes
		r.GET("/source_types", SourceTypeList, append(listMiddleware, middleware.LoggerFields)...)
		r.GET("/source_types/:id", SourceTypeGet, middleware.LoggerFields)
		r.GET("/source_types/:source_type_id/sources", SourceTypeListSource, tenancyWithListMiddleware...)

		// Red Hat Connector Connections
		r.GET("/rhc_connections", RhcConnectionList, tenancyWithListMiddleware...)
		r.GET("/rhc_connections/:id", RhcConnectionGetById, permissionMiddleware...)
		r.POST("/rhc_connections", RhcConnectionCreate, permissionMiddleware...)
		r.PATCH("/rhc_connections/:id", RhcConnectionEdit, append(permissionMiddleware, middleware.Notifier)...)
		r.DELETE("/rhc_connections/:id", RhcConnectionDelete, permissionMiddleware...)
		r.GET("/rhc_connections/:id/sources", RhcConnectionSourcesList, tenancyWithListMiddleware...)

		// GraphQL
		r.POST("/graphql", GraphQLQuery, tenancyMiddleware...)

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
		r.GET("/secrets/:id", InternalSecretGet, permissionMiddleware...)

		// Sources
		r.GET("/sources", InternalSourceList, permissionWithListMiddleware...)
		// Tenant translation endpoints.
		r.GET("/untranslated-tenants", GetUntranslatedTenants)
		r.POST("/translate-tenants", TranslateTenants)
	}
}
