package middleware

import (
	"github.com/labstack/echo/v4"
)

// DeprecationWarning returns a middleware that logs a deprecation warning
// for routes using the old internal API path format.
//
// The warning includes the new preferred path that should be used instead.
func DeprecationWarning(newPathFormat string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Logger().Warnf(
				"DEPRECATED: Path '%s' uses legacy internal API format. "+
					"Please migrate to new platform standard: '%s'. "+
					"Legacy format will be removed in a future release.",
				c.Request().URL.Path,
				newPathFormat,
			)

			// Add deprecation header to response
			c.Response().Header().Set("X-Deprecated-Path", "true")
			c.Response().Header().Set("X-Preferred-Path", newPathFormat)

			return next(c)
		}
	}
}
