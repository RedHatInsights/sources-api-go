package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var PSKS = conf.Psks

func permissionCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case c.Get("psk") != nil:
			psk, ok := c.Get("psk").(string)
			if !ok {
				return fmt.Errorf("error casting psk to string: %v", psk)
			}

			if !pskMatches(psk) {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: Incorrect PSK", "401"))
			}
		case c.Get("x-rh-identity") != nil:
			// TODO: Hit RBAC
		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}

func pskMatches(psk string) bool {
	return util.SliceContainsString(PSKS, psk)
}
