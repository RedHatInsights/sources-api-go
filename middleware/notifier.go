package middleware

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
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

		accountNumber, ok := c.Get("accountNumber").(string)
		if !ok {
			return fmt.Errorf("failed to cast account-number to string")
		}

		if emailNotificationInfo.PreviousAvailabilityStatus != emailNotificationInfo.CurrentAvailabilityStatus {
			return service.NotificationProducer.EmitAvailabilityStatusNotification(accountNumber, emailNotificationInfo)
		}

		return nil
	}
}
