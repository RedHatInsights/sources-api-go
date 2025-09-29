package fixtures

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

var userOwnership = "user"

var TestApplicationTypeData = []m.ApplicationType{
	{
		Id:          1,
		Name:        "app type one",
		DisplayName: "test app type",
	},
	{
		Id:          2,
		Name:        "app type two",
		DisplayName: "second test app type",
	},
	{
		Id:          5,
		Name:        "app type one123",
		DisplayName: "test app type",
	},
	{
		Id:          100,
		DisplayName: "app type without related sources",
	},
	{
		Id:                   3,
		DisplayName:          "app-studio",
		Name:                 "/insights/platform/app-studio",
		ResourceOwnership:    &userOwnership,
		SupportedSourceTypes: []byte(`["bitbucket", "dockerhub", "github", "gitlab", "quay"]`),
	},
	{
		Id:                           4,
		DisplayName:                  "Cost Management",
		Name:                         "/insights/platform/cost-management",
		SupportedSourceTypes:         []byte(`["amazon", "azure", "google", "openshift", "ibm"]`),
		SupportedAuthenticationTypes: []byte(`{"amazon": ["arn", "arn-2", "arn-3"], "azure": ["azure-auth", "azure-auth-2"]}`),
	},
	{
		Id:                           6,
		DisplayName:                  "Cloud Meter",
		Name:                         "/insights/platform/cloud-meter",
		SupportedSourceTypes:         []byte(`["amazon", "azure", "google", "openshift", "ibm"]`),
		SupportedAuthenticationTypes: []byte(`{"amazon": ["arn", "arn-2", "arn-3"], "azure": ["azure-auth", "azure-auth-2"]}`),
	},
	{
		Id:                           7,
		DisplayName:                  "Image Builder",
		Name:                         "/insights/platform/image-builder",
		SupportedSourceTypes:         []byte(`["amazon"]`),
		SupportedAuthenticationTypes: []byte(`{"amazon": ["arn", "arn-2", "arn-3"]}`),
	},
}
