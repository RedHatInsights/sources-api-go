package middleware

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/labstack/echo/v4"
)

/*
	This middleware intercepts a superkey-related source on its way through the
	stack and handles whether the requested resource is superkey related.

	If it is then we will queue up a job that sends the request over to the
	worker (to delete the resources in amazon), wait 15 seconds, then destroy
	the actual resources.
*/
func SuperKeyDestroySource(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return err
		}

		s := dao.GetSourceDao(&id)

		if s.IsSuperkey(id) {
			// TODO: queue up superkey delete job for source
			return c.NoContent(http.StatusAccepted)
		}

		return next(c)
	}
}

func SuperKeyDestroyApplication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return err
		}

		a := dao.GetApplicationDao(&id)

		if a.IsSuperkey(id) {
			// TODO: queue up superkey delete job for application
			return c.NoContent(http.StatusAccepted)
		}

		return next(c)
	}
}
