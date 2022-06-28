package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/RedHatInsights/rbac-client-go"
	"github.com/RedHatInsights/sources-api-go/config"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var (
	psks            = config.Get().Psks
	bypassRbac      = config.Get().BypassRbac
	rbacClient Rbac = &RbacClient{client: rbac.NewClient(os.Getenv("RBAC_URL"), "sources")}
)

/*
	Takes the information stored in the context and returns a 401 if we do not
	have authorization to perform "write" things such as POST/PATCH/DELETE.

	1. Checks for PSK (if present) and if it is there and matches any of the
	   PSKs we approve, lets it through.

	2. Sends the x-rh-identity header off to rbac to get an ACL list, and
	   returns whether or not it contains the correct `sources:*:*` permission.
*/
func PermissionCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case bypassRbac:
			c.Logger().Debugf("Skipping authorization check -- disabled in ENV")
		case c.Get(h.PSK) != nil:
			psk, ok := c.Get(h.PSK).(string)
			if !ok {
				return fmt.Errorf("error casting psk to string: %v", c.Get(h.PSK))
			}

			if !pskMatches(psk) {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: Incorrect PSK", "401"))
			}

		case c.Get(h.XRHID) != nil:
			// first check the identity (already parsed) to see if it contains
			// the system key and if it does do some extra checks to authorize
			// based on some internal rules (operator + satellite)
			identity, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
			if !ok {
				return fmt.Errorf("error casting identity to struct: %+v", c.Get("identity"))
			}

			// checking to see if we're going to change the results since
			// system-auth is treated completely differently than
			// org_admin/rbac/psk
			if identity.Identity.System != nil {
				// system-auth only allows GET and POST requests.
				method := c.Request().Method
				if method != http.MethodGet && method != http.MethodPost && method != http.MethodDelete {
					c.Response().Header().Set("Allow", "GET, POST, DELETE")
					return c.JSON(http.StatusMethodNotAllowed, util.ErrorDoc("Method not allowed", "405"))
				}
				// Secondary check for delete - we could move this to middleware
				if method == http.MethodDelete && !certDeleteAllowed(c) {
					c.Response().Header().Set("Allow", "GET, POST")
					return c.JSON(http.StatusMethodNotAllowed, util.ErrorDoc("Method not allowed", "405"))
				}

				// basically we're checking whether cn or cluster_id is set in
				// the system section of the header, if it is then this request
				// can go through (but only if it's a POST)
				//
				// we're returning early because this is easier than a goto.
				switch {
				case identity.Identity.System["cluster_id"] != nil:
					return next(c)
				case identity.Identity.System["cn"] != nil:
					return next(c)
				default:
					return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: system authorization only supports cn/cluster_id authorization", "401"))
				}
			}

			// otherwise, ship the xrhid off to rbac and check access rights.
			rhid, ok := c.Get(h.XRHID).(string)
			if !ok {
				return fmt.Errorf("error casting x-rh-identity to string: %v", c.Get("x-rh-identity"))
			}

			allowed, err := rbacClient.Allowed(rhid)
			if err != nil {
				return fmt.Errorf("error hitting rbac: %v", err)
			}

			if !allowed {
				return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Unauthorized Action: Missing RBAC permissions", "401"))
			}

		default:
			return c.JSON(http.StatusUnauthorized, util.ErrorDoc("Authentication required by either [x-rh-identity] or [x-rh-sources-psk]", "401"))
		}

		return next(c)
	}
}

func certDeleteAllowed(c echo.Context) bool {

	//Limit to "sources" endpoint - further filtering done by source handler
	m := regexp.MustCompile(`/sources/\d+$`)
	allowed := m.MatchString(c.Request().URL.Path)
	return allowed
}

func pskMatches(psk string) bool {
	return util.SliceContainsString(psks, psk)
}

type Rbac interface {
	Allowed(string) (bool, error)
}

type RbacClient struct {
	client rbac.Client
}

// fetches an access list from RBAC based on RBAC_URL and returns whether or not
// the xrhid has the `sources:*:*` permission
func (r *RbacClient) Allowed(xrhid string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acl, err := r.client.GetAccess(ctx, xrhid, "")
	if err != nil {
		return false, err
	}

	return acl.IsAllowed("sources", "*", "*"), nil
}
