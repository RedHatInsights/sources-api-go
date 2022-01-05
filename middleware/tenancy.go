package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

/*
	Parses all authorization related things into request context, notably:

	1. 'psk' -> x-rh-sources-psk

	2. 'identity' -> parsed version of identity header as a XRHID struct

	3. 'x-rh-identity' -> raw version (b64 encoded) of the identity header


	Returns a 401 if we cannot authorize the request from the required headers.
	For example if we have a psk but no account-number, or if we have neither
	headers when required.
*/
func Tenancy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// set psk + raw xrhid (if present) always - we'll need them later.
		if c.Request().Header.Get("x-rh-sources-psk") != "" {
			c.Set("psk", c.Request().Header.Get("x-rh-sources-psk"))
		}
		if c.Request().Header.Get("x-rh-identity") != "" {
			c.Set("x-rh-identity", c.Request().Header.Get("x-rh-identity"))
		}

		switch {
		case c.Request().Header.Get("x-rh-sources-account-number") != "":
			accountNumber := c.Request().Header.Get("x-rh-sources-account-number")
			c.Logger().Debugf("Looking up Tenant ID for account number %v", accountNumber)
			t, err := dao.GetOrCreateTenantID(accountNumber)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, util.ErrorDoc("Failed to get or create tenant for request", "500"))
			}
			c.Set("tenantID", *t)

		case c.Request().Header.Get("x-rh-identity") != "":
			idRaw, err := base64.StdEncoding.DecodeString(c.Request().Header.Get("x-rh-identity"))
			if err != nil {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc(fmt.Sprintf("Error decoding Identity: %v", err), "401"))
			}

			var jsonData identity.XRHID
			err = json.Unmarshal(idRaw, &jsonData)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("x-rh-identity header does not contain valid JSON", "401"))
			}

			// store the parsed header for later usage.
			c.Set("identity", jsonData)

			if jsonData.Identity.AccountNumber == "" {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Account number not present in x-rh-identity", "401"))
			}

			c.Logger().Debugf("Looking up Tenant ID for account number %v", jsonData.Identity.AccountNumber)
			t, err := dao.GetOrCreateTenantID(jsonData.Identity.AccountNumber)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, util.ErrorDoc(fmt.Sprintf("Failed to get or create tenant for request: %s", err.Error()), "500"))
			}
			c.Set("tenantID", *t)

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}
