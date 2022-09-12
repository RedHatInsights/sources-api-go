package middleware

import (
	"fmt"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func Notifier(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			return err
		}
		emailNotificationInfo, ok := c.Get("emailNotificationInfo").(*m.EmailNotificationInfo)
		if emailNotificationInfo == nil || !ok {
			return fmt.Errorf("unable to find emailNotificationInfo instance in middleware")
		}

		xRhIdentity, ok := c.Get(h.ParsedIdentityKey).(*identity.XRHID)
		if !ok {
			return fmt.Errorf("failed to fetch the identity header")
		}

		if emailNotificationInfo.PreviousAvailabilityStatus != emailNotificationInfo.CurrentAvailabilityStatus {
			return service.EmitAvailabilityStatusNotification(&xRhIdentity.Identity, emailNotificationInfo, "notifier-middleware")
		}

		return nil
	}
}
