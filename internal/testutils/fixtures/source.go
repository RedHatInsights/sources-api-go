package fixtures

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

var (
	uid1 = "5eebe172-7baa-4280-823f-19e597d091e9"
	uid2 = "31b5338b-685d-4056-ba39-d00b4d7f19cc"
)

var TestSourceData = []m.Source{
	{
		ID:           1,
		Name:         "Source1",
		SourceTypeID: 1,
		TenantID:     1,
		Uid:          &uid1,
	},
	{
		ID:           2,
		Name:         "Source2",
		SourceTypeID: 1,
		TenantID:     1,
		Uid:          &uid2,
	},
}
