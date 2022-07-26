package dao

import (
	"context"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

type RequestParams struct {
	TenantID *int64
	UserID   *int64
	ctx      context.Context
}

func NewRequestParamsFromContext(c echo.Context) (*RequestParams, error) {
	tenantId, err := util.GetTenantFromEchoContext(c)
	if err != nil {
		return nil, err
	}

	userID, err := util.GetUserFromEchoContext(c)
	if err != nil {
		return nil, err
	}

	var ctx context.Context
	if c.Request().Context() != nil {
		ctx = c.Request().Context()
	}

	return &RequestParams{TenantID: &tenantId, UserID: userID, ctx: ctx}, nil
}
