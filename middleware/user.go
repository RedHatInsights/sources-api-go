package middleware

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func UserCatcher(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantId, ok := c.Get(h.TENANTID).(int64)
		if !ok {
			return fmt.Errorf("failed to pull tenant from request")
		}

		var userIDFromContext string

		switch {
		case c.Get(h.PSK_USER) != nil:
			userID, ok := c.Get(h.PSK_USER).(string)
			if !ok {
				return fmt.Errorf("failed to pull user id from request")
			}

			userIDFromContext = userID

		case c.Get(h.PARSED_IDENTITY) != nil:
			xRhIdentity, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
			if !ok {
				return fmt.Errorf("failed to fetch the identity header")
			}

			userIDFromContext = xRhIdentity.Identity.User.UserID
		}

		if userIDFromContext != "" {
			user, err := dao.GetUserDao(&tenantId).FindOrCreate(userIDFromContext)
			if err != nil {
				return fmt.Errorf("unable to find or create user %v: %v", userIDFromContext, err)
			}

			c.Set(h.USERID, user.Id)
		}

		return next(c)
	}
}
