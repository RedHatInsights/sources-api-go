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
		xRhIdentity, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
		if !ok {
			return fmt.Errorf("failed to fetch the identity header")
		}

		tenantId, ok := c.Get(h.TENANTID).(int64)
		if !ok {
			return fmt.Errorf("failed to pull tenant from request")
		}

		userID := xRhIdentity.Identity.User.UserID
		if userID != "" {
			user, err := dao.GetUserDao(&tenantId).FindOrCreate(userID)
			if err != nil {
				return fmt.Errorf("unable to find or create user %v: %v", userID, err)
			}

			c.Set(h.USERID, user.Id)
		}

		return next(c)
	}
}
