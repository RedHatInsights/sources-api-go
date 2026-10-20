package fixtures

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

var (
	port                = 80
	defaultValueSource1 = true
	defaultValueSource2 = false
	scheme1             = "http"
	host1               = "openshift.example.com"
	path1               = "/"
	scheme2             = "https"
	host2               = "tower.example.com"
	path2               = "/"
)

var TestEndpointData = []m.Endpoint{
	{
		ID:                 1,
		SourceID:           1,
		TenantID:           1,
		Port:               &port,
		Default:            &defaultValueSource1,
		Scheme:             &scheme1,
		Host:               &host1,
		Path:               &path1,
		AvailabilityStatus: "unavailable",
	},
	{
		ID:                 2,
		SourceID:           1,
		TenantID:           1,
		Port:               &port,
		Default:            &defaultValueSource2,
		Scheme:             &scheme2,
		Host:               &host2,
		Path:               &path2,
		AvailabilityStatus: "unavailable",
	},
	{
		ID:                 300,
		SourceID:           2,
		TenantID:           1,
		Port:               &port,
		Default:            &defaultValueSource2,
		Scheme:             &scheme2,
		Host:               &host2,
		Path:               &path2,
		AvailabilityStatus: "unavailable",
	},
}
