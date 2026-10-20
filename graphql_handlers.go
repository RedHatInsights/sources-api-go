package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/graph"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	"github.com/labstack/echo/v4"
)

var (
	// the graphql handler server - initialized in the init() function below
	srv *handler.Server
	// the wrapped wrapper function for graphql
	wrapper echo.HandlerFunc
)

func init() {
	// setup the graphQL server
	srv = handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	srv.AddTransport(transport.POST{})

	// only set up introspection if we're not on stage/prod
	if os.Getenv("SOURCES_ENV") != "stage" && os.Getenv("SOURCES_ENV") != "prod" {
		srv.Use(extension.Introspection{})
	}

	wrapper = echo.WrapHandler(srv)
}

func GraphQLQuery(c echo.Context) error {
	requestParams, err := dao.NewRequestParamsFromContext(c)
	if requestParams == nil || err != nil {
		return fmt.Errorf("unable to process user id or tenant id from request")
	}

	// locking the initial sourceID mutex in order to load sources before
	// fetching any and all subresources
	sourceIdMutex := sync.Mutex{}
	sourceIdMutex.Lock()

	// store the `RequestData` we need for this request - this is the way we can
	// store certain data about the request for usage in the resolvers. kind of
	// like the graphql request context.
	c.SetRequest(
		c.Request().WithContext(context.WithValue(
			c.Request().Context(),
			graph.RequestData{},
			&graph.RequestData{
				// the current tenant
				TenantID: *requestParams.TenantID,
				UserID:   requestParams.UserID,
				// using a buffered channel so it does not block whens ending if
				// the count wasn't requested. it will be GC'd when the request
				// is done.
				CountChan: make(chan int, 1),
				// mutexes to ensure we load up all the source's
				// subresources _one time_
				ApplicationMutex:    &sync.Mutex{},
				EndpointMutex:       &sync.Mutex{},
				AuthenticationMutex: &sync.Mutex{},
				SourceMutex:         &sourceIdMutex,
			},
		)),
	)

	return wrapper(c)
}
