package main

import (
	"fmt"

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
	c.Set("resource", model.ToEvent())
}
