package fixtures

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/datatypes"
)

var TestApplicationData = []m.Application{
	{
		ID:                1,
		Extra:             datatypes.JSON("{\"extra\": true}"),
		ApplicationTypeID: 1,
		SourceID:          1,
		TenantID:          1,
	},
	{
		ID:                2,
		Extra:             datatypes.JSON("{\"extra\": false}"),
		ApplicationTypeID: 1,
		SourceID:          2,
		TenantID:          1,
	},
	{
		ID:                3,
		Extra:             datatypes.JSON("{\"extra\": false}"),
		ApplicationTypeID: 2,
		SourceID:          1,
		TenantID:          1,
	},
	{
		ID:                4,
		Extra:             datatypes.JSON("{\"extra\": false}"),
		ApplicationTypeID: 1,
		SourceID:          4,
		TenantID:          1,
	},
	{
		ID:                5,
		Extra:             datatypes.JSON("{\"extra\": false}"),
		ApplicationTypeID: 2,
		SourceID:          4,
		TenantID:          1,
	},
}
