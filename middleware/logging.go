package middleware

import (
	"context"

	l "github.com/RedHatInsights/sources-api-go/logger"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

func LoggerFields(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		uri := c.Request().RequestURI
		method := c.Request().Method
		agent := c.Request().UserAgent()
		acct := c.Get(h.AccountNumber)
		orgid := c.Get(h.OrgID)
		uuid := c.Get(h.InsightsRequestID)
		edgeId := c.Get(h.EdgeRequestID)

		baseFields := logrus.Fields{
			"uri":            uri,
			"method":         method,
			"account_number": acct,
			"org_id":         orgid,
			"user_agent":     agent,
			"request_id":     uuid,
			"edge_id":        edgeId,
		}

		baseLoggerEntry := l.Log.WithFields(baseFields)

		c.Set("logger", baseLoggerEntry)
		// holy cow echo makes this ugly. Setting a value on the actual net/http request's context
		c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), l.EchoLogger{}, baseLoggerEntry)))

		return next(c)
	}
}
