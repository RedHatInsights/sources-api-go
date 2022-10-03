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
		Id:   3,
		Name: "bitbucket",
	},
	{
		Id:   4,
		Name: "amazon123",
	},
	{
		Id:   100,
		Name: "source type without sources in fixtures",
	},
	{
		Id:   5,
		Name: "satellite",
	},
}
