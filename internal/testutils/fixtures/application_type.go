package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

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
		Id:          100,
		DisplayName: "app type without related sources",
	},
}
