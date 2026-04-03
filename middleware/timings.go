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

		// write the header before the response is written because we can't
		// write a header in middleware on the way out.
		c.Response().Before(func() {
			stopwatch := time.Since(begin)
			c.Response().Header().Set("X-Request-Duration-Ms", strconv.FormatInt(stopwatch.Milliseconds(), 10))
		})

		// continue down the stack...
		err := next(c)

		// log the total latency of the request on the way out
		latency := time.Since(begin)
		fields := make(logrus.Fields, 5)
		fields["latency"] = latency
		fields["latency_human"] = latency.String()

		if v := c.Get(h.InsightsRequestID); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["request_id"] = s
			}
		}

		if _, ok := fields["request_id"]; !ok {
			if v := c.Request().Header.Get(h.InsightsRequestID); v != "" {
				fields["request_id"] = v
			}
		}

		if v := c.Get(h.EdgeRequestID); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["edge_id"] = s
			}
		}

		if _, ok := fields["edge_id"]; !ok {
			if v := c.Request().Header.Get(h.EdgeRequestID); v != "" {
				fields["edge_id"] = v
			}
		}

		if entry, ok := c.Get("logger").(*logrus.Entry); ok {
			entry.WithFields(fields).Info()
		} else {
			logrus.WithFields(fields).Info()
		}

		return err
	}
}
