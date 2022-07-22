package util

import (
	"errors"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
)

func GetUserFromEchoContext(c echo.Context) (*int64, error) {
	userValue := c.Get(h.USERID)
	if userValue == nil {
		return nil, nil
	}

	if userId, ok := userValue.(int64); ok {
		if userId == 0 {
			return nil, nil
		}

		return &userId, nil
	} else {
		return nil, errors.New("the user was provided in an invalid format")
	}
}

// GetTenantFromEchoContext tries to extract the tenant from the echo context. If the "tenantID" is missing from the
// context, then a default value and nil are returned as the int64 and error values.
func GetTenantFromEchoContext(c echo.Context) (int64, error) {
	tenantValue := c.Get(h.TENANTID)

	// If no tenant is found in the context, that shouldn't imply an error.
	if tenantValue == nil {
		return 0, nil
	}

	// If the tenant is present, though, we must check that it is valid.
	if tenantId, ok := tenantValue.(int64); ok {
		if tenantId < 1 {
			return 0, errors.New("incorrect tenant value provided")
		}

		return tenantId, nil
	} else {
		return 0, errors.New("the tenant was provided in an invalid format")
	}
}
