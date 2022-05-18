package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/RedHatInsights/sources-api-go/graph"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

var (
	// the graphql handler server - initialized in the init() function below
	srv *handler.Server
	// the wrapped wrapper function for graphql
	wrapper echo.HandlerFunc

	// this is the uri to hit "old" sources api, initialized in the init() function
	legacyUri string
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

	// set the default host/port to what we'll see in k8s environments. Override
	// this by setting these values locally
	legacyHost := os.Getenv("RAILS_HOST")
	if legacyHost == "" {
		legacyHost = "sources-api-svc"
	}

	legacyPort := os.Getenv("RAILS_PORT")
	if legacyPort == "" {
		legacyPort = "8000"
	}

	legacyUri = "http://" + legacyHost + ":" + legacyPort + "/api/sources/v3.1/graphql"
}

func GraphQLQuery(c echo.Context) error {
	tenant, err := getTenantFromEchoContext(c)
	if err != nil {
		return err
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
				TenantID: tenant,
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

// this handler proxies the graphql request body + headers included over to the
// legacy rails side.
//
// it will probably sit here for quite a while - basically until we decide to
// implement graphql on the golang side which is a very low priority.
func ProxyGraphqlToLegacySources(c echo.Context) error {
	l.Log.Debugf("Proxying /graphql to [%v]", legacyUri)

	// set up a context with the parent as the request - limiting execution time to 10 seconds
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// create a request - passing the body right along
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, legacyUri, c.Request().Body)
	if err != nil {
		return err
	}

	// fetch the headers to forward along
	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	for _, h := range forwardableHeaders {
		req.Header.Add(h.Key, string(h.Value))
	}

	// add the content-type header, rails won't pick up the body otherwise
	req.Header.Add("Content-Type", "application/json")

	// run the request!
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return c.JSONBlob(resp.StatusCode, bytes)
}
