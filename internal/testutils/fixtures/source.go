package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestSourceData = []m.Source{
	{
		ID:           1,
		Name:         "Source1",
		SourceTypeID: 1,
		TenantID:     1,
	},
	{
		ID:           2,
		Name:         "Source2",
		SourceTypeID: 1,
		TenantID:     1,
	},
}
