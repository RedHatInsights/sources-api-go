package dao

import (
	"context"
	sourcesEcho "github.com/RedHatInsights/sources-api-go/util/echo"
	"github.com/labstack/echo/v4"
)

type RequestParams struct {
	TenantID *int64
	UserID   *int64
	ctx      context.Context
}

func NewRequestParamsFromContext(c echo.Context) (*RequestParams, error) {
	tenantId, err := sourcesEcho.GetTenantFromEchoContext(c)
	if err != nil {
		return nil, err
	}

	userID, err := sourcesEcho.GetUserFromEchoContext(c)
	if err != nil {
		return nil, err
	}

	var ctx context.Context
	switch {
	// if we wanted to override the context - pull that instead of the request's
	// context (which usually has a deadline we're trying to get around)
	case c.Get("override_context") != nil:
		var ok bool
		ctx, ok = c.Get("override_context").(context.Context)
		if !ok {
			c.Logger().Warn("Failed to pull overridden context")
		}
	case c.Request().Context() != nil:
		ctx = c.Request().Context()
	}

	return &RequestParams{TenantID: &tenantId, UserID: userID, ctx: ctx}, nil
}
