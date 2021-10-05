package middleware

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func HandleErrors(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			errorLog := util.ErrorLog{Logger: c.Logger(), Status: "500"}
			errorLog.Message = fmt.Sprintf("Internal Server Error: %v", err.Error())
			return c.JSON(
				http.StatusInternalServerError,
				errorLog.ErrorDocument(),
			)
		}
		return err
	}
}
