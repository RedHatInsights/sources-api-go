package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/util"
)

var keyRegex = regexp.MustCompile(`^filter\[(\w+)](\[\w*]|$)`)

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
				matches := keyRegex.FindAllStringSubmatch(key, -1)

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

func ParsePagination(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := setFromQueryWithDefault(c, "limit", 100)
		if err != nil {
			return c.JSON(http.StatusBadRequest, util.ErrorDoc("error parsing limit", "400"))
		}

		err = setFromQueryWithDefault(c, "offset", 0)
		if err != nil {
			return c.JSON(http.StatusBadRequest, util.ErrorDoc("error parsing offset", "400"))
		}

		return next(c)
	}
}

func setFromQueryWithDefault(c echo.Context, name string, def int) error {
	if c.QueryParam(name) != "" {
		val, err := strconv.Atoi(c.QueryParam(name))
		if err != nil {
			return err
		}

		c.Set(name, val)
	} else {
		c.Set(name, def)
	}

	return nil
}
