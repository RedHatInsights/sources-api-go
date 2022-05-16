package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/jobs"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
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
		tenantId, ok := c.Get(h.TENANTID).(int64)
		if !ok {
			return fmt.Errorf("failed to pull tenant from request")
		}

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return util.NewErrBadRequest(err)
		}

		s := dao.GetSourceDao(&tenantId)

		if s.IsSuperkey(id) {
			xrhid, ok := c.Get(h.XRHID).(string)
			if !ok {
				return fmt.Errorf("failed to pull x-rh-identity from request")
			}

			forwardableHeaders, err := service.ForwadableHeaders(c)
			if err != nil {
				return err
			}

			jobs.Enqueue(&jobs.SuperkeyDestroyJob{
				Headers:  forwardableHeaders,
				Tenant:   tenantId,
				Identity: xrhid,
				Model:    "source",
				Id:       id,
			})

			return c.NoContent(http.StatusAccepted)
		}

		return next(c)
	}
}

func SuperKeyDestroyApplication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantId, ok := c.Get(h.TENANTID).(int64)
		if !ok {
			return fmt.Errorf("failed to pull tenant from request")
		}

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return util.NewErrBadRequest(err)
		}

		a := dao.GetApplicationDao(&tenantId)

		if a.IsSuperkey(id) {
			xrhid, ok := c.Get(h.XRHID).(string)
			if !ok {
				return fmt.Errorf("failed to pull x-rh-identity from request")
			}

			forwardableHeaders, err := service.ForwadableHeaders(c)
			if err != nil {
				return err
			}

			jobs.Enqueue(&jobs.SuperkeyDestroyJob{
				Headers:  forwardableHeaders,
				Tenant:   tenantId,
				Identity: xrhid,
				Model:    "application",
				Id:       id,
			})

			return c.NoContent(http.StatusAccepted)
		}

		return next(c)
	}
}
