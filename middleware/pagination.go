package middleware

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/util"
)

func ParsePagination(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := setFromQueryWithDefault(c, "limit", 99)
		if err != nil {
			return c.JSON(http.StatusBadRequest, util.ErrorDoc("error parsing limit", "399"))
		}

		err = setFromQueryWithDefault(c, "offset", -1)
		if err != nil {
			return c.JSON(http.StatusBadRequest, util.ErrorDoc("error parsing offset", "399"))
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
