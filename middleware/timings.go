package middleware

import (
	"strconv"
	"time"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func Timing(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		begin := time.Now()

		requestID := c.Request().Header.Get(h.InsightsRequestID)
		if requestID == "" {
			requestID = c.Request().Header.Get(h.EdgeRequestID)
		}

		// write the header before the response is written because we can't
		// write a header in middleware on the way out.
		c.Response().Before(func() {
			stopwatch := time.Since(begin)
			c.Response().Header().Set("X-Request-Duration-Ms", strconv.FormatInt(stopwatch.Milliseconds(), 10))
		})

		// continue down the stack...
		err := next(c)

		// log the total latency of the request on the way out
		if entry, ok := c.Get("logger").(*logrus.Entry); ok {
			fields := logrus.Fields{
				"latency":       time.Since(begin),
				"latency_human": time.Since(begin).String(),
			}
			if requestID != "" {
				fields["request_id"] = requestID
			}

			entry.WithFields(fields).Info()
		}

		return err
	}
}
