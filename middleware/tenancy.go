package middleware

import (
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
		switch {
		case c.Get("psk-account") != nil:
			accountNumber, ok := c.Get("psk-account").(string)
			if !ok {
				return fmt.Errorf("failed to cast account-number to string")
			}

			c.Logger().Debugf("Looking up Tenant ID for account number %v", accountNumber)

			tenantDao := dao.GetTenantDao()
			tenantId, err := tenantDao.GetOrCreateTenantID(&identity.Identity{AccountNumber: accountNumber})
			if err != nil {
				return fmt.Errorf("failed to get or create tenant for request: %s", err)
			}

			c.Set("accountNumber", accountNumber)
			c.Set("tenantID", tenantId)

		case c.Get("identity") != nil:
			identity, ok := c.Get("identity").(identity.XRHID)
			if !ok {
				return fmt.Errorf("invalid identity structure received")
			}

			if identity.Identity.AccountNumber == "" && identity.Identity.OrgID == "" {
				return fmt.Errorf("account number or OrgId not present in x-rh-identity")
			}

			c.Logger().Debugf("[org_id: %s][account_number: %s] Looking up Tenant ID", identity.Identity.OrgID, identity.Identity.AccountNumber)

			tenantDao := dao.GetTenantDao()
			tenantId, err := tenantDao.GetOrCreateTenantID(&identity.Identity)
			if err != nil {
				return fmt.Errorf("failed to get or create tenant for request: %s", err)
			}

			c.Set("accountNumber", identity.Identity.AccountNumber)
			c.Set("tenantID", tenantId)

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}
