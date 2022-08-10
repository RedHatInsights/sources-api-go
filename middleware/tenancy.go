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
		id, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
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

		// Store the EBS account number and the OrgId in the context for easier usage later.
		c.Set(h.ACCOUNT_NUMBER, id.Identity.AccountNumber)
		c.Set(h.ORGID, id.Identity.OrgID)

		c.Logger().Debugf("[org_id: %s][account_number: %s] Looking up Tenant ID", id.Identity.OrgID, id.Identity.AccountNumber)

		tenantDao := dao.GetTenantDao()
		tenantId, err := tenantDao.GetOrCreateTenantID(&id.Identity)
		if err != nil {
			return fmt.Errorf("failed to get or create tenant for request: %s", err)
		}

		c.Set(h.TENANTID, tenantId)

		return next(c)
	}
}
