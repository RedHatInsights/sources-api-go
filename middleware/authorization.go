package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/rbac"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v5"
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

			case c.Get(h.XRHID) != nil:
				// first check the identity (already parsed) to see if it contains
				// the system key and if it does do some extra checks to authorize
				// based on some internal rules (operator + satellite)
				id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
				if !ok {
					return fmt.Errorf("error casting identity to struct: %+v", c.Get(h.ParsedIdentity))
				}

				// For system based authentications, we need to make sure that
				// the "Cluster ID" and the "Common name" fields are set in the
				// identity header.
				//
				// We also allow a subset of the HTTP methods when using this
				// type of authentications.
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

					// At this point the request is properly authenticated, so
					// there is no need to call RBAC.
					return next(c)
				}

				// For other types of authentications, we need to forward the
				// "x-rh-identity" header to RBAC to check if the principal
				// is authorized to perform the call.
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
				return c.JSON(http.StatusUnauthorized, util.NewErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
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

// isMethodAllowedForCertificateAuthentication returns true when the request's
// method is in the allowed list of methods for certificate based authenticated
// requests.
func isMethodAllowedForCertificateBasedAuthentication(method string) bool {
	return method == http.MethodGet || method == http.MethodPost || method == http.MethodDelete
}

// pskMatches returns true if the given PSK is in the list of allowed PSKs.
func pskMatches(authorizedPsks []string, psk string) bool {
	return util.SliceContainsString(authorizedPsks, psk)
}

// isUsingCertificateBasedAuthentication returns true if the given identity
// contains certificate details which could be used to perform a
// certificate-based authentication.
func isUsingCertificateBasedAuthentication(id *identity.XRHID) bool {
	// Check that the "System" struct is not "nil", and also make sure that it
	// is not empty or defined with empty values.
	//
	// The latter might happen if the header's JSON contents contain an empty
	// '"system": {}' object.
	return id.Identity.System != nil && *id.Identity.System != (identity.System{})
}
