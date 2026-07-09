package middleware

import (
	"fmt"
	"net/http"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/rbac"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// InternalPermissionCheck is similar to PermissionCheck but designed for the new
// /internal/sources/v{1,2} endpoints. It does NOT require PSK for cross-cluster
// calls (e.g., from crc* to hcc* clusters) and relies on x-rh-identity header or
// certificate-based authentication instead.
//
// This supports the new platform standard where internal APIs follow
// /internal/{APP}/{VERSION} format.
//
// Authentication methods supported:
//   - x-rh-identity header (certificate-based or regular)
//   - Bypass RBAC (for development/testing)
//
// Authentication methods NOT supported (compared to legacy PermissionCheck):
//   - PSK (pre-shared key) - intentionally excluded for new internal paths
func InternalPermissionCheck(bypassRbac bool, rbacClient rbac.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch {
			case bypassRbac:
				c.Logger().Debugf("Skipping authorization check -- disabled in ENV")

			case c.Get(h.XRHID) != nil:
				// Check if using certificate-based authentication
				id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
				if !ok {
					return fmt.Errorf("error casting identity to struct: %+v", c.Get(h.ParsedIdentity))
				}

				// For system/certificate-based authentications
				if isUsingCertificateBasedAuthentication(id) {
					// Make sure that the incoming system-authenticated request
					// is using the allowed method.
					method := c.Request().Method
					if !isMethodAllowedForCertificateBasedAuthentication(method) {
						c.Response().Header().Set("Allow", "GET, POST, DELETE")
						return c.JSON(http.StatusMethodNotAllowed, util.NewErrorDoc("Method not allowed", "405"))
					}

					// Make sure that the deletion operation is allowed.
					if method == http.MethodDelete && !certDeleteAllowed(c) {
						c.Response().Header().Set("Allow", "GET, POST")
						return c.JSON(http.StatusMethodNotAllowed, util.NewErrorDoc("Method not allowed", "405"))
					}

					// The "Cluster ID" and the "Common name" fields of the
					// certificate must have been specified in the header.
					if id.Identity.System.ClusterId == "" && id.Identity.System.CommonName == "" {
						return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: system authorization only supports cn/cluster_id authorization", "401"))
					}

					// At this point the request is properly authenticated
					return next(c)
				}

				// For other types of authentications (user-based), call RBAC
				rhid, ok := c.Get(h.XRHID).(string)
				if !ok {
					return fmt.Errorf(`authorization failed. The given "x-rh-identity" header is not a string: %v`, c.Get(h.XRHID))
				}

				allowed, err := rbacClient.Allowed(rhid)
				if err != nil {
					return fmt.Errorf("authorization failed. Unable to contact RBAC: %w", err)
				}

				if !allowed {
					return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: Missing RBAC permissions", "401"))
				}

			default:
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication required by [x-rh-identity] header", "401"))
			}

			return next(c)
		}
	}
}
