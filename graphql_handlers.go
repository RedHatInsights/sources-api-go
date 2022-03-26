package main

import (
	"context"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/RedHatInsights/sources-api-go/graph"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
)

var (
	// the graphql handler server - initialized in the init() function below
	srv *handler.Server
	// the wrapped handler function for graphql
	h echo.HandlerFunc
)

func init() {
	// setup the graphQL server
	srv = handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	srv.AddTransport(transport.POST{})

	// only set up introspection if we're not on stage/prod
	if os.Getenv("SOURCES_ENV") != "stage" && os.Getenv("SOURCES_ENV") != "prod" {
		srv.Use(extension.Introspection{})
	}

	h = echo.WrapHandler(srv)
}

func GraphQLQuery(c echo.Context) error {
	tenant, err := getTenantFromEchoContext(c)
	if err != nil {
		return err
	}

	// storing the tenant ID on the request context because the echo context's
	// store is just a map[string]interface{} not an actual context. Annoying.
	c.SetRequest(
		c.Request().WithContext(context.WithValue(
			c.Request().Context(),
			m.Tenant{},
			&m.Tenant{Id: tenant},
		)),
	)

	return h(c)
}
