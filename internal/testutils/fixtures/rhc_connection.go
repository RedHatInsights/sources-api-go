package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestRhcConnectionData = []m.RhcConnection{
	{
		ID:                 1,
		RhcId:              "a",
		AvailabilityStatus: "available",
	},
	{
		ID:                 2,
		RhcId:              "b",
		AvailabilityStatus: "available",
	},
	{
		ID:                 3,
		RhcId:              "c",
		AvailabilityStatus: "unavailable",
	},
}
