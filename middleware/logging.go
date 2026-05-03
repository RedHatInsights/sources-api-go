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
		sourceID := c.Param("source_id")
		applicationID := c.Param("application_id")

		authenticationID := c.Param("uid")
		if authenticationID == "" {
			authenticationID = c.Param("application_authentication_id")
		}

		baseFields := make(logrus.Fields, 10)
		baseFields["uri"] = uri
		baseFields["method"] = method
		baseFields["account_number"] = acct
		baseFields["org_id"] = orgid
		baseFields["user_agent"] = agent
		baseFields["request_id"] = uuid
		baseFields["edge_id"] = edgeId

		if sourceID != "" {
			baseFields["source_id"] = sourceID
		}

		if applicationID != "" {
			baseFields["application_id"] = applicationID
		}

		if authenticationID != "" {
			baseFields["authentication_id"] = authenticationID
		}

		baseLoggerEntry := l.Log.WithFields(baseFields)

		c.Set("logger", baseLoggerEntry)
		// holy cow echo makes this ugly. Setting a value on the actual net/http request's context
		c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), l.EchoLogger{}, baseLoggerEntry)))

		return next(c)
	}
}
