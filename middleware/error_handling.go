package middleware

import (
	"net/http"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func logErrorWithContextFields(c echo.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{"error": err, "panic": r}).Error("panic while logging error with context fields")
		}
	}()

	fields := make(logrus.Fields, 5)
	fields["error"] = err

	hasRequestID := false

	if v := c.Get(h.InsightsRequestID); v != nil {
		if s, ok := v.(string); ok && s != "" {
			fields["request_id"] = s
			hasRequestID = true
		}
	}

	if !hasRequestID {
		if v := c.Request().Header.Get(h.InsightsRequestID); v != "" {
			fields["request_id"] = v
			hasRequestID = true
		}
	}

	if !hasRequestID {
		if v := c.Request().Header.Get(h.EdgeRequestID); v != "" {
			fields["request_id"] = v
		}
	}

	if p := c.Param("source_id"); p != "" {
		fields["source_id"] = p
	}

	if p := c.Param("application_id"); p != "" {
		fields["application_id"] = p
	}

	if p := c.Param("uid"); p != "" {
		fields["authentication_id"] = p
	} else if p := c.Param("application_authentication_id"); p != "" {
		fields["authentication_id"] = p
	}

	if entry, ok := c.Get("logger").(*logrus.Entry); ok {
		entry.WithFields(fields).Error(err)
	} else {
		logrus.WithFields(fields).Error(err)
	}
}

func HandleErrors(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			var (
				statusCode int
				message    interface{}
			)

			switch err.(type) {
			case util.ErrNotFound:
				statusCode = http.StatusNotFound
				message = util.NewErrorDoc(err.Error(), "404")
			case util.ErrBadRequest:
				statusCode = http.StatusBadRequest
				message = util.NewErrorDoc(err.Error(), "400")
			default:
				uuid, ok := c.Get(h.InsightsRequestID).(string)
				if !ok {
					uuid = ""
				}

				statusCode = http.StatusInternalServerError
				message = util.ErrorDocWithRequestId("Internal Server Error", "500", uuid)

				logErrorWithContextFields(c, err)
			}

			return c.JSON(statusCode, message)
		}

		return nil
	}
}
