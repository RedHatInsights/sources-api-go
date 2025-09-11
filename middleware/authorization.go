package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/RedHatInsights/sources-api-go/config"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/rbac"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
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
func PermissionCheck(bypassRbac bool, authorizedPsks []string, authorizedJWTSubjects []config.AuthorizedJWTSubject, rbacClient rbac.Client) echo.MiddlewareFunc {
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

			case c.Get(h.JWTIssuer) != nil && c.Get(h.JWTSubject) != nil:
				jwtIssuer, ok := c.Get(h.JWTIssuer).(string)
				if !ok {
					return fmt.Errorf("error casting jwt issuer to string: %v", c.Get(h.JWTIssuer))
				}

				jwtSubject, ok := c.Get(h.JWTSubject).(string)
				if !ok {
					return fmt.Errorf("error casting jwt subject to string: %v", c.Get(h.JWTSubject))
				}

				if !jwtClaimsMatches(authorizedJWTSubjects, jwtIssuer, jwtSubject) {
					return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: JWT issuer/subject not authorized", "401"))
				}

			case c.Get(h.XRHID) != nil:
				// first check the identity (already parsed) to see if it contains
				// the system key and if it does do some extra checks to authorize
				// based on some internal rules (operator + satellite)
				id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
				if !ok {
					return fmt.Errorf("error casting identity to struct: %+v", c.Get(h.ParsedIdentity))
				}

				// checking to see if we're going to change the results since
				// system-auth is treated completely differently than
				// org_admin/rbac/psk
				if id.Identity.System != (&identity.System{}) {
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
					if id.Identity.System != nil {
						if id.Identity.System.ClusterId != "" || id.Identity.System.CommonName != "" {
							return next(c)
						} else {
							return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Unauthorized Action: system authorization only supports cn/cluster_id authorization", "401"))
						}
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
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication required by either [x-rh-identity], [x-rh-sources-psk] or [Authorization: Bearer <token>]", "401"))
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

// jwtClaimsMatches returns true if the given JWT issuer/subject pair matches any authorized combination.
func jwtClaimsMatches(authorizedJWTSubjects []config.AuthorizedJWTSubject, jwtIssuer, jwtSubject string) bool {
	for _, authorized := range authorizedJWTSubjects {
		if authorized.Issuer == jwtIssuer && authorized.Subject == jwtSubject {
			return true
		}
	}

	return false
}
