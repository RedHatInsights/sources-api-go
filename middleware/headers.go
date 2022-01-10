package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

/*
   Parse the required headers for processing this request. Currently this
   involves _three_ major headers:

   1. `x-rh-identity`: contains the account number and various other information
      about the request. This is set by 3scale.

   2. `x-rh-sources-psk`: a pre-shared-key (psk) which is used internally to
      authenticate from within the CRC cluster. This is checked against a list
      of known keys which are set in vault, if it matches any of them the
      request is authorized.

    3. `x-rh-sources-account-number`: used with a PSK to access a certain
       account. Only accessible from within the CRC cluster.
*/
func ParseHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// the PSK related headers - just storing them as raw strings.
		if c.Request().Header.Get("x-rh-sources-psk") != "" {
			c.Set("psk", c.Request().Header.Get("x-rh-sources-psk"))
		}

		if c.Request().Header.Get("x-rh-sources-account-number") != "" {
			c.Set("psk-account", c.Request().Header.Get("x-rh-sources-account-number"))
		}

		// parsing the base64-encoded identity header if present
		if c.Request().Header.Get("x-rh-identity") != "" {
			// store it raw first.
			c.Set("x-rh-identity", c.Request().Header.Get("x-rh-identity"))

			idRaw, err := base64.StdEncoding.DecodeString(c.Request().Header.Get("x-rh-identity"))
			if err != nil {
				return fmt.Errorf("error decoding Identity: %v", err)
			}

			var id identity.XRHID
			err = json.Unmarshal(idRaw, &id)
			if err != nil {
				return fmt.Errorf("x-rh-identity header does not contain valid JSON")
			}

			// store the parsed header for later usage.
			c.Set("identity", id)

			// store whether or not this a cert-auth based request
			if id.Identity.System != nil && id.Identity.System["cn"] != nil {
				c.Set("cert-auth", true)
			}
		}

		return next(c)
	}
}
