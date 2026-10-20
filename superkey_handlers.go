package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/jobs"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

// enqueueSuperKeyDelete is the shared logic for async superkey deletes.
// When a resource (source or application) is superkey-managed, the delete
// must be handled asynchronously: we enqueue a job that tells the superkey
// worker to tear down the cloud resources first, then cascade-deletes the
// DB records after a short delay.
//
// Returns 202 Accepted to indicate the delete has been accepted but will
// complete asynchronously.
func enqueueSuperKeyDelete(c echo.Context, model string, id int64) error {
	tenantId, ok := c.Get(h.TenantID).(int64)
	if !ok {
		return fmt.Errorf("failed to pull tenant from request")
	}

	xrhid, ok := c.Get(h.XRHID).(string)
	if !ok {
		return fmt.Errorf("failed to pull x-rh-identity from request")
	}

	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	jobs.Enqueue(&jobs.SuperkeyDestroyJob{
		Headers:  forwardableHeaders,
		Tenant:   tenantId,
		Identity: xrhid,
		Model:    model,
		Id:       id,
	})

	return c.NoContent(http.StatusAccepted)
}
