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
		SourceID:          1,
		TenantID:          1,
	},
}
