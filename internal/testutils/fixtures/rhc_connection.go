package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestRhcConnectionData = []m.RhcConnection{
	{
		ID:    1,
		RhcId: "a",
		AvailabilityStatus: m.AvailabilityStatus{
			AvailabilityStatus: "available",
		},
	},
	{
		ID:    2,
		RhcId: "b",
		AvailabilityStatus: m.AvailabilityStatus{
			AvailabilityStatus: "available",
		},
	},
	{
		ID:    3,
		RhcId: "c",
		AvailabilityStatus: m.AvailabilityStatus{
			AvailabilityStatus: "unavailable",
		},
	},
}
