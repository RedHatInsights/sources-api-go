package middleware

import (
	"fmt"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
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

		if c.Request().Header.Get(h.AccountNumber) != "" {
			c.Set(h.AccountNumber, c.Request().Header.Get(h.AccountNumber))
		}

		if c.Request().Header.Get(h.OrgID) != "" {
			c.Set(h.OrgID, c.Request().Header.Get(h.OrgID))
		}

		if c.Request().Header.Get(h.PSKUserID) != "" {
			c.Set(h.PSKUserID, c.Request().Header.Get(h.PSKUserID))
		}

		if c.Request().Header.Get(h.EdgeRequestID) != "" {
			c.Set(h.EdgeRequestID, c.Request().Header.Get(h.EdgeRequestID))
		}

		if c.Request().Header.Get(h.InsightsRequestID) != "" {
			c.Set(h.InsightsRequestID, c.Request().Header.Get(h.InsightsRequestID))
		}

		// id is the XrhId struct we will store in the context. The idea is: if no "x-rh-identity" header has been
		// received, generate an identity struct from the given "OrgId" and "EBS account number". If the "x-rh-identity"
		// header is present, simply decode it and use the decided identity.
		var id *identity.XRHID

		xRhIdentityRaw := c.Request().Header.Get(h.XRHID)
		if xRhIdentityRaw == "" {
			generatedIdentity := util.GeneratedXRhIdentity(c.Request().Header.Get(h.AccountNumber), c.Request().Header.Get(h.OrgID))

			// Store the raw, base64 encoded "xRhIdentity" string.
			c.Set(h.XRHID, generatedIdentity)

			// Generate the identity which we will store in the context.
			genId, err := util.ParseXRHIDHeader(generatedIdentity)
			if err != nil {
				return fmt.Errorf("could not generate the x-rh-identity structure: %w", err)
			}

			id = genId
		} else {
			xRhIdentity, err := util.ParseXRHIDHeader(xRhIdentityRaw)
			if err != nil {
				return fmt.Errorf("could not extract identity from header: %w", err)
			}

			// Store the raw identity header to forward it latter.
			c.Set(h.XRHID, xRhIdentityRaw)

			// Store whether this a cert-auth based request.
			if xRhIdentity.Identity.System.CommonName != "" {
				c.Set("cert-auth", true)
			}

			id = xRhIdentity
		}

		c.Set(h.ParsedIdentity, id)

		return next(c)
	}
}
