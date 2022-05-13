package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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

			// Use the whole "XRHID" struct for consistency, since many other parts in the code are expecting the
			// "identity" context variable to have this struct.
			id := &identity.XRHID{
				Identity: identity.Identity{
					AccountNumber: accountNumber,
				},
			}

			tenantDao := dao.GetTenantDao()
			tenantId, err := tenantDao.GetOrCreateTenantID(&id.Identity)
			if err != nil {
				return fmt.Errorf("failed to get or create tenant for request: %s", err)
			}

			c.Set("identity", id)
			c.Set("tenantID", tenantId)

		case c.Get("psk-org-id") != nil:
			orgId, ok := c.Get("psk-org-id").(string)
			if !ok {
				return errors.New("failed to cast orgId to string")
			}

			c.Logger().Debugf(`[org_id: %s] Looking up Tenant ID`, orgId)

			// Use the whole "XRHID" struct for consistency, since many other parts in the code are expecting the
			// "identity" context variable to have this struct.
			id := &identity.XRHID{
				Identity: identity.Identity{
					OrgID: orgId,
				},
			}

			tenantDao := dao.GetTenantDao()
			tenantId, err := tenantDao.GetOrCreateTenantID(&id.Identity)
			if err != nil {
				return fmt.Errorf("failed to get or create tenant for request: %s", err)
			}

			c.Set("identity", id)
			c.Set("tenantID", tenantId)

		case c.Get("identity") != nil:
			identity, ok := c.Get("identity").(*identity.XRHID)
			if !ok {
				return fmt.Errorf("invalid identity structure received")
			}

			// Check that we received at least an account number or an org ID.
			// In the case of receiving an identity with an OrgId, but without an AccountNumber, log it since we need
			// to be on the lookout for these anemic tenants. There might be services that still only support using
			// EbsAccount numbers, and might not work otherwise.
			if identity.Identity.AccountNumber == "" {
				if identity.Identity.OrgID == "" {
					return fmt.Errorf("account number or OrgId not present in x-rh-identity")
				} else {
					log.Warnf(`[org_id: %s] potential anemic tenant found`, identity.Identity.OrgID)
				}
			}

			c.Logger().Debugf("[org_id: %s][account_number: %s] Looking up Tenant ID", identity.Identity.OrgID, identity.Identity.AccountNumber)

			tenantDao := dao.GetTenantDao()
			tenantId, err := tenantDao.GetOrCreateTenantID(&identity.Identity)
			if err != nil {
				return fmt.Errorf("failed to get or create tenant for request: %s", err)
			}

			c.Set("tenantID", tenantId)

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}
