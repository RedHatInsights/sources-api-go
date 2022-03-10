package fixtures

import (
	"encoding/json"

	m "github.com/RedHatInsights/sources-api-go/model"
)

var TestAuthenticationData = []m.Authentication{
	{
		ID:       "611a8a38-f434-4e62-bda0-78cd45ffae5b",
		TenantID: TestTenantData[0].Id,
		SourceID: 1,
	},
}

var TestAuthenticationVaultData = map[string]interface{}{
	"authtype":            "test",
	"availability_status": "available",
	"availability_error":  "",
	"resource_type":       "Application",
	"resource_id":         "1",
	"extra":               map[string]interface{}{},
	"name":                "OpenShift",
	"source_id":           "1",
	"username":            "testusr",
}

var TestAuthenticationVaultMetaData = map[string]interface{}{
	"created_time":    "2022-01-10T14:58:02.850209Z",
	"custom_metadata": "2022-01-10T14:58:02.850209Z",
	"deletion_time":   nil,
	"destroyed":       false,
	"version":         json.Number("2"),
	"extra":           map[string]interface{}{},
	"name":            "OpenShift",
	"source_id":       "1",
	"username":        "testusr",
}
