package fixtures

import m "github.com/RedHatInsights/sources-api-go/model"

var TestSourceRhcConnectionData = []m.SourceRhcConnection{
	{
		SourceId:        TestSourceData[0].ID,
		RhcConnectionId: TestRhcConnectionData[0].ID,
		TenantId:        TestTenantData[0].Id,
	},
	{
		SourceId:        TestSourceData[0].ID,
		RhcConnectionId: TestRhcConnectionData[1].ID,
		TenantId:        TestTenantData[0].Id,
	},
	{
		SourceId:        TestSourceData[1].ID,
		RhcConnectionId: TestRhcConnectionData[0].ID,
		TenantId:        TestTenantData[0].Id,
	},
	{
		SourceId:        TestSourceData[1].ID,
		RhcConnectionId: TestRhcConnectionData[2].ID,
		TenantId:        TestTenantData[0].Id,
	},
}
