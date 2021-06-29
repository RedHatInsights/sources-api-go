package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/util"
)

func HandleErrors(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			return c.JSON(
				http.StatusInternalServerError,
				util.ErrorDoc(fmt.Sprintf("Internal Server Error: %v", err.Error()), "500"),
			)
		}
		return err
	}
}
