package util

import (
	"errors"

	l "github.com/RedHatInsights/sources-api-go/logger"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// SourcesContext is a wrapper around the echo.Context struct so we can override
// methods provided by the echo library
type SourcesContext struct{ echo.Context }

// overriding the Logger method so we can inject custom fields into the logger.
// The first time this method is called (for the request) some parsing is done
// and stored back on the echo context. From there the fields will be present on
// every invocation of (*echo.Context).Logger().Debug/Info/Error/Warn
func (sc *SourcesContext) Logger() echo.Logger {
	// check if we've populated the base log field first, if so return that.
	// Otherwise just return the base echo "empty" logge
	if sc.Get("logger") != nil {
		if entry, ok := sc.Get("logger").(*logrus.Entry); ok {
			return l.EchoLogger{Entry: entry}
		}
	}

	return sc.Echo().Logger
}

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
