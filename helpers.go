package main

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func getFilters(c echo.Context) ([]util.Filter, error) {
	var filters []util.Filter
	var ok bool

	filterVal := c.Get("filters")
	if filters, ok = filterVal.([]util.Filter); !ok {
		return nil, fmt.Errorf("failed to pull filters from request")
	}

	return filters, nil
}

func getLimitAndOffset(c echo.Context) (int, int, error) {
	var limit, offset int
	var ok bool

	limitVal := c.Get("limit")
	if limit, ok = limitVal.(int); !ok {
		return 0, 0, util.NewErrBadRequest("failed to parse limit")
	}

	if limit < 1 {
		return 0, 0, util.NewErrBadRequest("limit is less than 1")
	}

	offsetVal := c.Get("offset")
	if offset, ok = offsetVal.(int); !ok {
		return 0, 0, util.NewErrBadRequest("failed to parse offset")
	}

	if offset < 0 {
		return 0, 0, util.NewErrBadRequest("offset is less than 0")
	}

	return limit, offset, nil
}

func setNotificationForAvailabilityStatus(c echo.Context, previousStatus string, resource m.EmailNotification) {
	c.Set("emailNotificationInfo", resource.ToEmail(previousStatus))
}

func setEventStreamResource(c echo.Context, model m.Event) {
	// get the model type we're raising the event for
	// 1. Strip the pointer symbol
	// 2. Strip the package prefix ("model.")
	t := reflect.TypeOf(model)
	m := strings.TrimPrefix(t.String(), "*")
	m = strings.TrimPrefix(m, "model.")

	// get the event type that just happened based on the http request
	event := ""
	switch c.Request().Method {
	case http.MethodPost:
		event = ".create"
	case http.MethodPatch:
		event = ".update"
	case http.MethodDelete:
		event = ".destroy"
	default:
		c.Logger().Warnf("Unsupported request type, middleware should probably not be here: %v", c.Request().Method)
	}

	c.Set("event_type", m+event)
	c.Set("resource", model)
}

// getTenantFromEchoContext tries to extract the tenant from the echo context. If the "tenantID" is missing from the
// context, then a default value and nil are returned as the int64 and error values.
func getTenantFromEchoContext(c echo.Context) (int64, error) {
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

func getAccountNumberFromEchoContext(c echo.Context) (string, error) {
	id, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
	if !ok {
		return "", fmt.Errorf("failed to pull identity from context")
	}

	return id.Identity.AccountNumber, nil
}
