package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestMetaDataData = []m.MetaData{
	{
		ID:                1,
		ApplicationTypeID: 1,
		Type:              "AppMetaData",
	},
	{
		ID:                2,
		ApplicationTypeID: 1,
		Type:              "AppMetaData",
	},
	{
		ID:                3,
		ApplicationTypeID: 2,
		Type:              "AppMetaData",
	},
}
