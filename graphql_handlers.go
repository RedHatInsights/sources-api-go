package main

import (
	"context"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/RedHatInsights/sources-api-go/graph"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
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

	// store the `RequestData` we need for this request - this is the way we can
	// store certain data about the request for usage in the resolvers. kind of
	// like the graphql request context.
	c.SetRequest(
		c.Request().WithContext(context.WithValue(
			c.Request().Context(),
			graph.RequestData{},
			&graph.RequestData{
				TenantID: tenant,
				// using a buffered channel so it does not block whens ending if the count wasn't requested. it will be GC'd when the request is done.
				CountChan: make(chan int, 1),
			},
		)),
	)

	return h(c)
}
