package fixtures

import (
	"encoding/json"

	m "github.com/RedHatInsights/sources-api-go/model"
)

var availabilityStatus = "available"

var TestAuthenticationData = []m.Authentication{
	{
		ID:                 "611a8a38-f434-4e62-bda0-78cd45ffae5b",
		DbID:               1,
		TenantID:           TestTenantData[0].Id,
		SourceID:           1,
		ResourceType:       "Application",
		ResourceID:         1,
		AvailabilityStatus: &availabilityStatus,
	},
	{
		ID:                 "82e1a1b6-a136-11ec-b909-0242ac120002",
		DbID:               2,
		TenantID:           TestTenantData[0].Id,
		SourceID:           2,
		ResourceType:       "Endpoint",
		ResourceID:         1,
		AvailabilityStatus: &availabilityStatus,
	},
	{
		ID:                 "24127a2a-c4db-11ec-9d64-0242ac120002",
		DbID:               3,
		TenantID:           TestTenantData[0].Id,
		SourceID:           2,
		ResourceType:       "Source",
		ResourceID:         1,
		AvailabilityStatus: &availabilityStatus,
	},
	{
		ID:                 "1683629d-b830-4542-8842-308525cb7004",
		DbID:               5,
		TenantID:           TestTenantData[0].Id,
		SourceID:           2,
		ResourceType:       "Application",
		ResourceID:         4,
		AvailabilityStatus: &availabilityStatus,
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
