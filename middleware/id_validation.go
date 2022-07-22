package middleware

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var idValidationRegex = regexp.MustCompile(`^\d+$`)

// IdValidation takes all the parameters which end with "id" and checks for their validity. Returns a bad request if
// they're not valid.
func IdValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		for _, param := range c.ParamNames() {
			if strings.HasSuffix(param, "id") {
				err := validateId(c, param)
				if err != nil {
					return util.NewErrBadRequest(err)
				}
			}
		}

		return next(c)
	}
}

// UuidValidation checks if the UUID parameter is valid. Returns a bad request if it isn't.
func UuidValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := validateUuid(c)
		if err != nil {
			return util.NewErrBadRequest(err)
		}

		return next(c)
	}
}

// validateId checks if the ID stored in specified context parameter isn't empty, is a valid integer and that it is
// greater than zero.
func validateId(c echo.Context, idParamName string) error {
	idRaw := c.Param(idParamName)

	if !idValidationRegex.MatchString(idRaw) {
		return errors.New("the provided ID must be a greater than zero, positive number")
	}

	// This check is required for the edge case of an out of range error.
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse the provided ID: %s", err)
	}

	if id == 0 {
		return errors.New("the provided ID must be greater than zero")
	}

	return nil
}

// validateUuid checks if the UUID is not empty.
func validateUuid(c echo.Context) error {
	uid := c.Param("uid")

	if uid == "" {
		return errors.New("the UUID cannot be empty or missing")
	}

	return nil
}
