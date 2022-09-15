package middleware

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// Tenancy is a middleware which makes sure the EBS account number or OrgId are present, and therefore, the request is
// properly authenticated. It sets the tenant ID on the context by looking in the database using the provided EBS
// account number or OrgId.
func Tenancy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, ok := c.Get(h.ParsedIdentityKey).(*identity.XRHID)
		if !ok {
			return fmt.Errorf("invalid identity structure received: %#v", id)
		}

		// Check that we received at least an account number or an org ID.
		// In the case of receiving an identity with an OrgId, but without an AccountNumber, log it since we need
		// to be on the lookout for these anemic tenants. There might be services that still only support using
		// EbsAccount numbers, and might not work otherwise.
		if id.Identity.AccountNumber == "" {
			if id.Identity.OrgID == "" {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("the ebs account number and the org id are missing", "401"))
			} else {
				c.Logger().Warnf(`[org_id: %s] potential anemic tenant found`, id.Identity.OrgID)
			}
		}

		c.Logger().Debugf("[org_id: %s][account_number: %s] Looking up Tenant ID", id.Identity.OrgID, id.Identity.AccountNumber)

		tenantDao := dao.GetTenantDao()
		tenant, err := tenantDao.GetOrCreateTenant(&id.Identity)
		if err != nil {
			c.Logger().Errorf("[identity struct: %v] unable to get or create the tenant: %w", err)

			return fmt.Errorf("failed to get or create tenant for request: %w", err)
		}

		// Update the identity struct with the tenancy data from the database.
		id.Identity.OrgID = tenant.OrgID
		id.Identity.AccountNumber = tenant.ExternalTenant
		c.Set(h.ParsedIdentityKey, id)

		// Store the ID, EBS account number and OrgId from what we've got in the database. Prior to this, we stored
		// the contents of the incoming headers, but this had a problem: if we only received an EBS account number, we
		// would only forward that account number even if we had a complementary OrgId number stored.
		//
		// This can cause issues with services we are integrated with —notifications, for example—, which will not
		// accept EBS account numbers anymore. However, we can deal with that by forwarding the OrgId too if we have it
		// stored in the database.
		c.Set(h.TenantIdKey, tenant.Id)
		c.Set(h.AccountNumber, tenant.ExternalTenant)
		c.Set(h.OrgIdKey, tenant.OrgID)

		return next(c)
	}
}
