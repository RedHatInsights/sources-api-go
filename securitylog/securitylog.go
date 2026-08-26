package securitylog

import (
	"net/http"
	"strconv"

	l "github.com/RedHatInsights/sources-api-go/logger"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/sirupsen/logrus"
)

// GetPrincipal extracts principal information from the echo context.
func GetPrincipal(c echo.Context) (orgID, userID, principalType string) {
	id, ok := c.Get(h.ParsedIdentity).(*identity.XRHID)
	if !ok || id == nil {
		return "", "", "anonymous"
	}

	orgID = id.Identity.OrgID

	switch {
	case id.Identity.User != nil && id.Identity.User.UserID != "":
		return orgID, id.Identity.User.UserID, "user"
	case id.Identity.ServiceAccount != nil && id.Identity.ServiceAccount.ClientId != "":
		return orgID, id.Identity.ServiceAccount.ClientId, "service_account"
	case id.Identity.System != nil && id.Identity.System.CommonName != "":
		return orgID, id.Identity.System.CommonName, "system"
	default:
		return orgID, "", "unknown"
	}
}

func securityFields(action, resourceType, resourceID, outcome, orgID, userID, principalType string) logrus.Fields {
	return logrus.Fields{
		"security_event": true,
		"action":         action,
		"resource_type":  resourceType,
		"resource_id":    resourceID,
		"outcome":        outcome,
		"principal": map[string]string{
			"org_id":  orgID,
			"user_id": userID,
			"type":    principalType,
		},
	}
}

// FormatID converts an int64 to string for use as resource_id.
func FormatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// LogCrud logs a CRUD security event, extracting the principal from echo.Context.
func LogCrud(c echo.Context, action, resourceType, resourceID, outcome string) {
	orgID, userID, principalType := GetPrincipal(c)
	l.Log.WithFields(securityFields(action, resourceType, resourceID, outcome, orgID, userID, principalType)).
		Info("[security_event: true]")
}

// LogStartup logs a process startup security event (EOI-5).
func LogStartup(outcome string) {
	l.Log.WithFields(logrus.Fields{
		"security_event": true,
		"action":         "STARTUP",
		"resource_type":  "process",
		"resource_id":    "sources-api-go",
		"outcome":        outcome,
		"principal": map[string]string{
			"type": "system",
		},
	}).Info("[security_event: true]")
}

// LogShutdown logs a process shutdown security event (EOI-5).
func LogShutdown(outcome, reason string) {
	fields := logrus.Fields{
		"security_event": true,
		"action":         "SHUTDOWN",
		"resource_type":  "process",
		"resource_id":    "sources-api-go",
		"outcome":        outcome,
		"principal": map[string]string{
			"type": "system",
		},
	}
	if reason != "" {
		fields["reason"] = reason
	}

	if outcome == "failure" {
		l.Log.WithFields(fields).Error("[security_event: true]")
	} else {
		l.Log.WithFields(fields).Info("[security_event: true]")
	}
}

// LogAuthFailure logs an authentication failure security event (EOI-7).
func LogAuthFailure(reason, remoteAddr string) {
	l.Log.WithFields(logrus.Fields{
		"security_event": true,
		"action":         "AUTH_FAILURE",
		"resource_type":  "authentication",
		"resource_id":    "",
		"outcome":        "failure",
		"reason":         reason,
		"remote_addr":    remoteAddr,
		"principal": map[string]string{
			"type": "anonymous",
		},
	}).Warn("[security_event: true]")
}

// LogAuthzFailure logs an authorization failure security event (EOI-8).
func LogAuthzFailure(c echo.Context, reason string) {
	orgID, userID, principalType := GetPrincipal(c)
	l.Log.WithFields(logrus.Fields{
		"security_event": true,
		"action":         "AUTHZ_FAILURE",
		"resource_type":  "authorization",
		"resource_id":    "",
		"outcome":        "failure",
		"reason":         reason,
		"principal": map[string]string{
			"org_id":  orgID,
			"user_id": userID,
			"type":    principalType,
		},
	}).Warn("[security_event: true]")
}

// IsMutatingMethod returns true for POST, PUT, PATCH, DELETE.
func IsMutatingMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}
