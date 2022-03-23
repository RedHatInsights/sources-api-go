package main

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

func ProxyGraphqlToLegacySources(c echo.Context) error {
	// set up a context with the parent as the request - limiting execution time to 10 seconds
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// create a request - using the body as the post body
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		"http://localhost:3002/api/sources/v3.1/graphql",
		c.Request().Body,
	)
	if err != nil {
		return err
	}

	// fetch the headers to forward along
	for _, h := range service.ForwadableHeaders(c) {
		req.Header.Add(h.Key, string(h.Value))
	}

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
