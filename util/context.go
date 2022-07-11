package util

import (
	"errors"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
)

func GetUserFromEchoContext(c echo.Context) (string, error) {
	userValue := c.Get(h.USERID)

	if userValue == nil {
		return "", nil
	}

	if userId, ok := userValue.(string); ok {
		if userId == "" {
			return "", errors.New("incorrect user value provided")
		}

		return userId, nil
	} else {
		return "", errors.New("the user was provided in an invalid format")
	}
}
