package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/middleware/oidc"
	"github.com/RedHatInsights/sources-api-go/rbac"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// Feature flag constants
const (
	FeatureFlagOIDCAuth = "sources.oidc_authentication"
)

// PermissionCheck takes the authentication information stored in the context and returns a "401 — Unauthorized" if the
// given request is not authorized to perform "write" operations such as "POST, PATCH and DELETE".
//
//   - When "bypassRbac" is "true", all the requests are authenticated and authorized.
//   - When using a "psk" in the request, the latter gets authorized if the PSK is in our list of authorized PSKs that
//     can send requests to Sources.
//   - Lastly, the requests that come with an "x-rh-identity" header must fulfill one of the following two conditions:
//   - The request has been authenticated with a certificate, and it's been sent with an allowed "GET", "POST" or
//     "DELETE" http verb. In the case that it's a "DELETE" request, it will only be authorized to perform that
//     operation on a subset of paths.
//   - The request is a regularly authenticated one, so we will call RBAC to verify that the principal that comes in
//     the header has the authorization to perform the operation in Sources.
func PermissionCheck(bypassRbac bool, authorizedPsks []string, rbacClient rbac.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch {
			case bypassRbac:
				c.Logger().Debugf("Skipping authorization check -- disabled in ENV")
			case c.Get(h.PSK) != nil:
				psk, ok := c.Get(h.PSK).(string)
				if !ok {
					return fmt.Errorf("error casting psk to string: %v", c.Get(h.PSK))
				}

				if !pskMatches(authorizedPsks, psk) {
					return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: Incorrect PSK", "401"))
				}

			case c.Get(h.JWTToken) != nil && service.FeatureEnabled(FeatureFlagOIDCAuth):
				// JWT-based authentication for S2S communication - prioritize over XRHID
				// This feature is gated behind the OIDC authentication feature flag
				jwtConfig := oidc.LoadJWTConfigFromGlobal()

				tokenString, ok := c.Get(h.JWTToken).(string)
				if !ok {
					return fmt.Errorf("error casting JWT token to string: %v", c.Get(h.JWTToken))
				}

				userID, err := oidc.ValidateJWTWithConfig(tokenString, jwtConfig)
				if err != nil {
					c.Logger().Debugf("JWT validation failed: %v", err)
					return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: Invalid JWT token", "401"))
				}

				// Store JWT user ID in context for user middleware
				c.Set(h.JWTUserID, userID)
				c.Logger().Debugf("JWT authentication successful for user: %s", userID)

			case c.Get(h.JWTToken) != nil && !service.FeatureEnabled(FeatureFlagOIDCAuth):
				// JWT token present but OIDC authentication feature flag is disabled
				// Log this and fall through to XRHID authentication
				c.Logger().Debugf("JWT token present but OIDC authentication is disabled via feature flag")
				fallthrough

			case c.Get(h.XRHID) != nil:
				// first check the identity (already parsed) to see if it contains
				// the system key and if it does do some extra checks to authorize
				// based on some internal rules (operator + satellite)
				id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
				if !ok {
					return fmt.Errorf("error casting identity to struct: %+v", c.Get(h.ParsedIdentity))
				}

				// If the identity is empty AND it's the generated default identity (no real auth provided),
				// fall through to default case
				if id.Identity.AccountNumber == "" && id.Identity.OrgID == "" && id.Identity.System == (identity.System{}) {
					// Check if this is the generated empty identity from ParseHeaders
					xrhidRaw, ok := c.Get(h.XRHID).(string)
					if !ok {
						return fmt.Errorf("error casting x-rh-identity to string: %v", c.Get(h.XRHID))
					}

					generatedEmptyIdentity := "eyJpZGVudGl0eSI6eyJvcmdfaWQiOiIiLCJpbnRlcm5hbCI6eyJvcmdfaWQiOiIifSwidXNlciI6eyJ1c2VybmFtZSI6IiIsImVtYWlsIjoiIiwiZmlyc3RfbmFtZSI6IiIsImxhc3RfbmFtZSI6IiIsImlzX2FjdGl2ZSI6ZmFsc2UsImlzX29yZ19hZG1pbiI6ZmFsc2UsImlzX2ludGVybmFsIjpmYWxzZSwibG9jYWxlIjoiIiwidXNlcl9pZCI6IiJ9LCJzeXN0ZW0iOnt9LCJhc3NvY2lhdGUiOnsiUm9sZSI6bnVsbCwiZW1haWwiOiIiLCJnaXZlbk5hbWUiOiIiLCJyaGF0VVVJRCI6IiIsInN1cm5hbWUiOiIifSwieDUwOSI6eyJzdWJqZWN0X2RuIjoiIiwiaXNzdWVyX2RuIjoiIn0sInR5cGUiOiJTeXN0ZW0iLCJhdXRoX3R5cGUiOiIifX0="
					if xrhidRaw == generatedEmptyIdentity {
						return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication required by either [x-rh-identity], [x-rh-sources-psk], or [Authorization: Bearer <token>]", "401"))
					}
				}

				// checking to see if we're going to change the results since
				// system-auth is treated completely differently than
				// org_admin/rbac/psk
				if id.Identity.System != (identity.System{}) {
					// system-auth only allows GET and POST requests.
					method := c.Request().Method
					if method != http.MethodGet && method != http.MethodPost && method != http.MethodDelete {
						c.Response().Header().Set("Allow", "GET, POST, DELETE")
						return c.JSON(http.StatusMethodNotAllowed, util.NewErrorDoc("Method not allowed", "405"))
					}
					// Secondary check for delete - we could move this to middleware
					if method == http.MethodDelete && !certDeleteAllowed(c) {
						c.Response().Header().Set("Allow", "GET, POST")
						return c.JSON(http.StatusMethodNotAllowed, util.NewErrorDoc("Method not allowed", "405"))
					}

					// basically we're checking whether cn or cluster_id is set in
					// the system section of the header, if it is then this request
					// can go through (but only if it's a POST)
					//
					// we're returning early because this is easier than a goto.
					if id.Identity.System.ClusterId != "" || id.Identity.System.CommonName != "" {
						return next(c)
					} else {
						return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: system authorization only supports cn/cluster_id authorization", "401"))
					}
				}

				// otherwise, ship the xrhid off to rbac and check access rights.
				rhid, ok := c.Get(h.XRHID).(string)
				if !ok {
					return fmt.Errorf("error casting x-rh-identity to string: %v", c.Get(h.XRHID))
				}

				allowed, err := rbacClient.Allowed(rhid)
				if err != nil {
					return fmt.Errorf("error hitting rbac: %v", err)
				}

				if !allowed {
					return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: Missing RBAC permissions", "401"))
				}

			default:
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication required by either [x-rh-identity], [x-rh-sources-psk], or [Authorization: Bearer <token>]", "401"))
			}

			return next(c)
		}
	}
}

// certDeleteAllowed returns true when the given "DELETE" request authenticated by a certificate has been sent to the
// paths which we allow those request to be sent for.
func certDeleteAllowed(c echo.Context) bool {
	//Limit to "sources" endpoint - further filtering done by source handler.
	return regexp.MustCompile(`/sources/\d+$`).MatchString(c.Request().URL.Path)
}

// pskMatches returns true if the given PSK is in the list of allowed PSKs.
func pskMatches(authorizedPsks []string, psk string) bool {
	return util.SliceContainsString(authorizedPsks, psk)
}
