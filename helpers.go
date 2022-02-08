package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
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
		return 0, 0, fmt.Errorf("failed to parse limit")
	}

	offsetVal := c.Get("offset")
	if offset, ok = offsetVal.(int); !ok {
		return 0, 0, fmt.Errorf("failed to parse offset")
	}

	return limit, offset, nil
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
	c.Set("resource", model.ToEvent())
}
