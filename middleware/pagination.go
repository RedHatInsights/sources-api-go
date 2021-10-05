package middleware

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func Pagination(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := parsePaginationIntoContext(c)
		if err != nil {
			return err
		}

		return next(c)
	}
}

func parsePaginationIntoContext(c echo.Context) error {
	if c.QueryParam("limit") != "" {
		val, err := strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			errorLog := util.ErrorLog{Logger: c.Logger(), LogMessage: err.Error(), Message: "error parsing limit"}
			return c.JSON(http.StatusBadRequest, errorLog.ErrorDocument())
		}

		c.Set("limit", val)
	} else {
		c.Set("limit", 100)
	}

	if c.QueryParam("offset") != "" {
		val, err := strconv.Atoi(c.QueryParam("offset"))
		if err != nil {
			errorLog := util.ErrorLog{Logger: c.Logger(), LogMessage: err.Error(), Message: "error parsing offset"}
			return c.JSON(http.StatusBadRequest, errorLog.ErrorDocument())
		}

		c.Set("offset", val)
	} else {
		c.Set("offset", 0)
	}

	return nil
}
