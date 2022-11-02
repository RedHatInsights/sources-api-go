package middleware

import (
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var BadQueryParams = []string{"limit", "offset", "sort_by"}

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
			filter := util.Filter{}

			// matching filter[subresource][field][operation]
			// we only support filtering on source type/application type
			if matches[1][0] == "source_type" || matches[1][0] == "application_type" {
				filter = util.Filter{
					Subresource: matches[1][0],
					Name:        matches[2][0],
					Value:       values,
				}

				// this will only be present if the user used a operation. it
				// defaults to "" which we use as eq
				if len(matches) == 4 {
					filter.Operation = matches[3][0]
				}
			} else {
				// matching filter[field][operation]
				filter = util.Filter{Name: matches[1][0], Value: values}
				// this will only be present if the user used a operation. it
				// defaults to "" which we use as eq
				if len(matches) == 3 {
					filter.Operation = matches[2][0]
				}
			}

			f = append(f, filter)
		} else if !util.SliceContainsString(BadQueryParams, key) {
			// This is to ensure backward compatibility with the Rails API. Any
			// query parameters that do not have `filter` or `sort_by` at the
			// beginning need to be treated as "raw" output filters. e.g. if
			// someone passes `id=25` we need to support that as equivalent to
			// `filter[id]=25`
			//
			// if the column doesn't exist PG will throw a sqlstate error -
			// which is expected.
			f = append(f, util.Filter{Name: key, Value: values})
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

	return &util.Filter{Operation: "sort_by", Value: []string{"id ASC"}}
}
