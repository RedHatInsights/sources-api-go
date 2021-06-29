package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/middleware"
)

func getFilters(c echo.Context) ([]middleware.Filter, error) {
	var filters []middleware.Filter
	var ok bool

	filterVal := c.Get("filters")
	if filters, ok = filterVal.([]middleware.Filter); !ok {
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
