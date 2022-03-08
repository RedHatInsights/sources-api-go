package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func Timing(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		begin := time.Now()

		c.Response().Before(func() {
			stopwatch := time.Since(begin)
			c.Response().Header().Set("X-Request-Duration-Ms", strconv.FormatInt(stopwatch.Milliseconds(), 10))
		})

		return next(c)
	}
}
