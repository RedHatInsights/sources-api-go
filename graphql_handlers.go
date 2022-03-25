package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/RedHatInsights/sources-api-go/graph"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

// this is the uri to hit "old" sources api, initialized in the init() function
var legacyUri string

// set the default host/port to what we'll see in k8s environments. Override
// this by setting these values locally
func init() {
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
	for _, h := range service.ForwadableHeaders(c) {
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

func GraphQLQuery(c echo.Context) error {
	tenant, err := getTenantFromEchoContext(c)
	if err != nil {
		return err
	}

	h := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{
			// injecting the tenant into the resolver
			Resolvers: &graph.Resolver{TenantID: tenant},
		}))

	return echo.WrapHandler(h)(c)
}
