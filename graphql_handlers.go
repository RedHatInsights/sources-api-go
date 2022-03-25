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

// the graphql handler server var - initialized in the init() function below
var h *handler.Server

func init() {
	// setup the graphQL server
	h = handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	h.AddTransport(transport.POST{})

	// only set up introspection if we're not on stage/prod
	if os.Getenv("SOURCES_ENV") != "stage" && os.Getenv("SOURCES_ENV") != "prod" {
		h.Use(extension.Introspection{})
	}
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

	// wrapping the http handler and calling it with the echo context
	return echo.WrapHandler(h)(c)
}
