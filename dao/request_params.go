package dao

import (
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

type RequestParams struct {
	TenantID *int64
	UserID   *int64
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

	return &RequestParams{TenantID: &tenantId, UserID: userID}, nil
}
