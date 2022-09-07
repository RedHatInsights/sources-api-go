package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestMetaDataData = []m.MetaData{
	{
		ID:                1,
		ApplicationTypeID: 1,
		Type:              m.APP_META_DATA,
	},
	{
		ID:                2,
		ApplicationTypeID: 1,
		Type:              m.APP_META_DATA,
	},
}
