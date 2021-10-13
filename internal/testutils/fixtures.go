package testutils

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/datatypes"
)

var TestTenantData = []m.Tenant{
	{Id: 1},
}

var TestSourceTypeData = []m.SourceType{
	{Id: 1, Name: "amazon"},
}

var TestApplicationTypeData = []m.ApplicationType{
	{Id: 1, DisplayName: "test app type"},
}

var TestSourceData = []m.Source{
	{ID: 1, Name: "Source1", SourceTypeID: 1, TenantID: 1},
	{ID: 2, Name: "Source2", SourceTypeID: 1, TenantID: 1},
}

var TestApplicationData = []m.Application{
	{ID: 1, Extra: datatypes.JSON("{\"extra\": true}"), ApplicationTypeID: 1, SourceID: 1, TenantID: 1},
	{ID: 2, Extra: datatypes.JSON("{\"extra\": false}"), ApplicationTypeID: 1, SourceID: 1, TenantID: 1},
}

var TestEndpointData = []m.Endpoint{
	{ID: 1, SourceID: 1, TenantID: 1},
	{ID: 2, SourceID: 1, TenantID: 1},
}

var TestMetaDataData = []m.MetaData{
	{ID: 1, ApplicationTypeID: 1, Type: "AppMetaData"},
	{ID: 2, ApplicationTypeID: 1, Type: "AppMetaData"},
}
