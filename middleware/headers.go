package middleware

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
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

		if c.Request().Header.Get("x-rh-sources-org-id") != "" {
			c.Set("psk-org-id", c.Request().Header.Get("x-rh-sources-org-id"))
		}

		// parsing the base64-encoded identity header if present
		if c.Request().Header.Get("x-rh-identity") != "" {
			// store it raw first.
			c.Set("x-rh-identity", c.Request().Header.Get("x-rh-identity"))

			xRhIdentity, err := util.ParseXRHIDHeader(c.Request().Header.Get("x-rh-identity"))
			if err != nil {
				return fmt.Errorf("could not extract identity from header: %s", err)
			}

			// store the parsed header for later usage.
			c.Set("identity", xRhIdentity)
			// store the psk to pass along in headers
			c.Set("psk-account", xRhIdentity.Identity.AccountNumber)

			// store whether or not this a cert-auth based request
			if xRhIdentity.Identity.System != nil && xRhIdentity.Identity.System["cn"] != nil {
				c.Set("cert-auth", true)
			}
		} else {
			dummyIdentity := util.XRhIdentityWithAccountNumber(c.Request().Header.Get("x-rh-sources-account-number"))

			// backup xrhid from account number (in case of psk auth)
			c.Set("x-rh-identity", dummyIdentity)

			// store the parsed header for later usage.
			id, err := util.ParseXRHIDHeader(dummyIdentity)
			if err != nil {
				return err
			}
			c.Set("identity", *id)
		}

		return next(c)
	}
}
