package middleware

import (
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func SortAndFilter(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		filters := parseFilter(c)
		if sort := parseSorting(c); sort != nil {
			filters = append(filters, *sort)
		}

		c.Set("filters", filters)
		return next(c)
	}
}

func parseFilter(c echo.Context) []util.Filter {
	f := make([]util.Filter, 0)
	for key, values := range c.QueryParams() {
		if strings.HasPrefix(key, "filter") {
			matches := util.FilterRegex.FindAllStringSubmatch(key, -1)

			filter := util.Filter{Name: matches[0][1], Value: values}
			if len(matches[0]) == 3 {
				filter.Operation = matches[0][2]
			}

			f = append(f, filter)
		}
	}

	return f
}

func parseSorting(c echo.Context) *util.Filter {
	for k, v := range c.QueryParams() {
		if k == "sort_by" {
			return &util.Filter{Operation: "sort_by", Value: v}
		}
	}

	return nil
}
