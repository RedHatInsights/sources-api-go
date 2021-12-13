package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/RedHatInsights/rbac-client-go"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var PSKS = conf.Psks

// TODO: move to middleware package after removing circular dependency
func permissionCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case conf.BypassRbac:
			c.Logger().Debugf("Skipping authorization check -- disabled in ENV")
		case c.Get("psk") != nil:
			psk, ok := c.Get("psk").(string)
			if !ok {
				return fmt.Errorf("error casting psk to string: %v", c.Get("psk"))
			}

			if !pskMatches(psk) {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: Incorrect PSK", "401"))
			}
		case c.Get("x-rh-identity") != nil:
			rhid, ok := c.Get("x-rh-identity").(string)
			if !ok {
				return fmt.Errorf("error casting x-rh-identity to string: %v", c.Get("x-rh-identity"))
			}

			allowed, err := rbacAllowed(rhid)
			if err != nil {
				return fmt.Errorf("error hitting rbac: %v", err)
			}

			if !allowed {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: Missing RBAC permissions", "401"))
			}
		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}

func pskMatches(psk string) bool {
	return util.SliceContainsString(PSKS, psk)
}

var r = rbac.NewClient(os.Getenv("RBAC_URL"), "sources")

func rbacAllowed(rhid string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acl, err := r.GetAccess(ctx, rhid, "")
	if err != nil {
		return false, err
	}

	return acl.IsAllowed("sources", "*", "*"), nil
}
