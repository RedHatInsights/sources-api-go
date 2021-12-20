package fixtures

import (
	"time"

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

var (
	CreatedAt, _        = time.Parse("2006-01-02T15:04:05.999Z", "2021-11-26 18:47:19 CET")
	UpdatedAt           = CreatedAt
	port                = 80
	defaultValueSource1 = true
	defaultValueSource2 = false
	scheme1             = "http"
	host1               = "openshift.example.com"
	path1               = "/"
	scheme2             = "https"
	host2               = "tower.example.com"
	path2               = "/"
)

var TestEndpointData = []m.Endpoint{
	{ID: 1, SourceID: 1, TenantID: 1, Port: &port, Default: &defaultValueSource1, Scheme: &scheme1, Host: &host1, Path: &path1, CreatedAt: CreatedAt, UpdatedAt: UpdatedAt},
	{ID: 2, SourceID: 1, TenantID: 1, Port: &port, Default: &defaultValueSource2, Scheme: &scheme2, Host: &host2, Path: &path2, CreatedAt: CreatedAt, UpdatedAt: UpdatedAt},
}

var TestMetaDataData = []m.MetaData{
	{ID: 1, ApplicationTypeID: 1, Type: "AppMetaData"},
	{ID: 2, ApplicationTypeID: 1, Type: "AppMetaData"},
}
