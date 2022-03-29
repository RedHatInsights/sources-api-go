package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestSourceTypeData = []m.SourceType{
	{
		Id:   1,
		Name: "amazon",
	},
	{
		Id:   2,
		Name: "google",
	},
	{
		Id:   100,
		Name: "source type without sources in fixtures",
	},
}
