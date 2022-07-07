package middleware

import (
	"fmt"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func ResourceOwnership(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		identity, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
		if !ok {
			return fmt.Errorf("invalid identity structure received")
		}

		c.Set(h.USERID, identity.Identity.User.UserID)
		return next(c)
	}
}
