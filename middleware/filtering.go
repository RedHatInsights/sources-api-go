package middleware

import (
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var filterRegex = regexp.MustCompile(`^filter\[(\w+)](\[\w*]|$)`)

type Filter struct {
	Name      string
	Operation string
	Value     []string
}

func ParseFilter(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
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

		c.Set("filters", f)

		return next(c)
	}
}
