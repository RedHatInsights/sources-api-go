package main

import (
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// handlerLogEntry returns the per-request *logrus.Entry set by middleware.LoggerFields when present.
// That entry already includes uri, method, account_number, org_id, user_agent, request_id, edge_id,
// and any of source_id, application_id, authentication_id from the route when those params exist
// (see middleware/logging.go). Use WithFields below to add tenant_id and resource ids; duplicate keys
// from the route are overwritten with the typed values from the handler when needed.
// If LoggerFields did not run, falls back to the root logger.
func handlerLogEntry(c echo.Context) *logrus.Entry {
	if entry, ok := c.Get("logger").(*logrus.Entry); ok {
		return entry
	}

	return logger.Log.WithFields(logrus.Fields{})
}
