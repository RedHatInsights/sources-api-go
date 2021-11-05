package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func HandleErrors(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			var statusCode int
			var message interface{}

			if errors.Is(err, util.ErrNotFoundEmpty) {
				statusCode = http.StatusNotFound
				message = util.ErrorDoc(err.Error(), "404")
			} else {
				statusCode = http.StatusInternalServerError
				message = util.ErrorDoc(fmt.Sprintf("Internal Server Error: %v", err.Error()), "500")
			}
			return c.JSON(statusCode, message)
		}

		return nil
	}
}
