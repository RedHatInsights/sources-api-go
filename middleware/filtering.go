package middleware

import (
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var filterRegex = regexp.MustCompile(`^filter\[(\w+)](\[\w*]|$)`)

// TODO: make the dao package not rely on this.
type Filter struct {
	Name      string
	Operation string
	Value     []string
}

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

func parseFilter(c echo.Context) []Filter {
	f := make([]Filter, 0)
	for key, values := range c.QueryParams() {
		if strings.HasPrefix(key, "filter") {
			matches := filterRegex.FindAllStringSubmatch(key, -1)

			filter := Filter{Name: matches[0][1], Value: values}
			if len(matches[0]) == 3 {
				filter.Operation = matches[0][2]
			}

			f = append(f, filter)
		}
	}

	return f
}

func parseSorting(c echo.Context) *Filter {
	for k, v := range c.QueryParams() {
		if k == "sort_by" {
			return &Filter{Operation: "sort_by", Value: v}
		}
	}

	return nil
}
