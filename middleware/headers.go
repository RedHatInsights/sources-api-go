package middleware

import (
	"fmt"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
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
		if c.Request().Header.Get(h.PSK) != "" {
			c.Set(h.PSK, c.Request().Header.Get(h.PSK))
		}

		if c.Request().Header.Get(h.ACCOUNT_NUMBER) != "" {
			c.Set(h.ACCOUNT_NUMBER, c.Request().Header.Get(h.ACCOUNT_NUMBER))
		}

		if c.Request().Header.Get(h.ORGID) != "" {
			c.Set(h.ORGID, c.Request().Header.Get(h.ORGID))
		}

		if c.Request().Header.Get(h.PSK_USER) != "" {
			c.Set(h.PSK_USER, c.Request().Header.Get(h.PSK_USER))
		}

		// parsing the base64-encoded identity header if present
		if c.Request().Header.Get(h.XRHID) != "" {
			// store it raw first.
			c.Set(h.XRHID, c.Request().Header.Get(h.XRHID))

			xRhIdentity, err := util.ParseXRHIDHeader(c.Request().Header.Get(h.XRHID))
			if err != nil {
				return fmt.Errorf("could not extract identity from header: %s", err)
			}

			// store the parsed header for later usage.
			c.Set(h.PARSED_IDENTITY, xRhIdentity)

			// store the account number + org_id for easy usage later
			c.Set(h.ACCOUNT_NUMBER, xRhIdentity.Identity.AccountNumber)
			c.Set(h.ORGID, xRhIdentity.Identity.OrgID)

			// store whether or not this a cert-auth based request
			if xRhIdentity.Identity.System.CommonName != "" {
				c.Set("cert-auth", true)
			}
		} else {
			dummyIdentity := util.GeneratedXRhIdentity(
				c.Request().Header.Get(h.ACCOUNT_NUMBER),
				c.Request().Header.Get(h.ORGID),
			)

			// backup xrhid from account number (in case of psk auth)
			c.Set(h.XRHID, dummyIdentity)

			// store the parsed header for later usage.
			id, err := util.ParseXRHIDHeader(dummyIdentity)
			if err != nil {
				return err
			}
			c.Set(h.PARSED_IDENTITY, id)
		}

		return next(c)
	}
}
