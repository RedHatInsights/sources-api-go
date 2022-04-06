package dao

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockSourceDao struct {
	Sources        []m.Source
	RelatedSources []m.Source
}

type MockApplicationDao struct {
	Applications []m.Application
}

type MockEndpointDao struct {
	Endpoints []m.Endpoint
}
type MockApplicationTypeDao struct {
	ApplicationTypes []m.ApplicationType
	Compatible       bool
}

type MockSourceTypeDao struct {
	SourceTypes []m.SourceType
}

type MockMetaDataDao struct {
	MetaDatas []m.MetaData
}

type MockRhcConnectionDao struct {
	RhcConnections        []m.RhcConnection
	RelatedRhcConnections []m.RhcConnection
}

type MockApplicationAuthenticationDao struct {
	ApplicationAuthentications []m.ApplicationAuthentication
}

type MockAuthenticationDao struct {
	Authentications []m.Authentication
}

func (src *MockSourceDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	var sources []m.Source

	switch object := primaryCollection.(type) {
	case m.SourceType:
		var sourceTypeExists bool
		for _, sourceType := range fixtures.TestSourceTypeData {
			if sourceType.Id == object.Id {
				sourceTypeExists = true
			}
		}

		// Source type doesn't exist = return Not Found err
		if !sourceTypeExists {
			return nil, 0, util.NewErrNotFound("source type")
		}

		// Source type exists = return sources subcollection
		for index, source := range src.Sources {
			if source.SourceTypeID == object.Id {
				sources = append(sources, src.Sources[index])
			}
		}

	case m.ApplicationType:
		var appTypeExists bool
		for _, appType := range fixtures.TestApplicationTypeData {
			if appType.Id == object.Id {
				appTypeExists = true
			}
		}

		// Application type doesn't exist = return Not Found err
		if !appTypeExists {
			return nil, 0, util.NewErrNotFound("application type")
		}

		// Application type exists = find applications with given app type
		// and save list of related source IDs
		var sourceIDs []int64
		for _, app := range fixtures.TestApplicationData {
			if app.ApplicationTypeID == object.Id {
				sourceIDs = append(sourceIDs, app.SourceID)
			}
		}

		// For each source ID find source
		for _, sourceID := range sourceIDs {
			for _, s := range src.Sources {
				if s.ID == sourceID {
					sources = append(sources, s)
				}
			}
		}

	default:
		return nil, 0, fmt.Errorf("unexpected primary collection type")
	}

	count := int64(len(sources))
	return sources, count, nil
}

func (src *MockSourceDao) List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(src.Sources))
	return src.Sources, count, nil
}

func (src *MockSourceDao) ListInternal(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(src.Sources))
	return src.Sources, count, nil
}

func (src *MockSourceDao) GetById(id *int64) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (src *MockSourceDao) Create(s *m.Source) error {
	src.Sources = append(src.Sources, *s)
	return nil
}

func (src *MockSourceDao) Update(s *m.Source) error {
	return nil
}

func (src *MockSourceDao) Delete(id *int64) (*m.Source, error) {
	for i, source := range src.Sources {
		if source.ID == *id {
			src.Sources = append(src.Sources[:i], src.Sources[i+1:]...)

			return &source, nil
		}
	}
	return nil, util.NewErrNotFound("source")
}

func (src *MockSourceDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

// NameExistsInCurrentTenant returns always false because it's the safe default in case the request gets validated
// in the tests.
func (src *MockSourceDao) NameExistsInCurrentTenant(name string) bool {
	return false
}

func (src *MockSourceDao) IsSuperkey(id int64) bool {
	return false
}

func (src *MockSourceDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (m *MockSourceDao) ListForRhcConnection(id *int64, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(m.RelatedSources))

	return m.RelatedSources, count, nil
}

func (m *MockSourceDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSourceDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) error {
	return nil
}

func (m *MockSourceDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (s *MockSourceDao) Pause(_ int64) error {
	return nil
}

func (s *MockSourceDao) Unpause(_ int64) error {
	return nil
}

func (a *MockApplicationTypeDao) List(limit int, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	count := int64(len(a.ApplicationTypes))
	return a.ApplicationTypes, count, nil
}

func (a *MockApplicationTypeDao) GetById(id *int64) (*m.ApplicationType, error) {
	for _, i := range a.ApplicationTypes {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("application type")
}

func (a *MockMetaDataDao) List(limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	count := int64(len(a.MetaDatas))
	return a.MetaDatas, count, nil
}

func (a *MockMetaDataDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	var appMetaDataList []m.MetaData

	for index, i := range a.MetaDatas {
		switch object := primaryCollection.(type) {
		case m.ApplicationType:
			if i.ApplicationTypeID == object.Id {
				appMetaDataList = append(appMetaDataList, a.MetaDatas[index])
			}
		default:
			return nil, 0, fmt.Errorf("unexpected primary collection type")
		}
	}
	count := int64(len(appMetaDataList))
	if count == 0 {
		return nil, count, util.NewErrNotFound("metadata")
	}

	return appMetaDataList, count, nil
}

func (a *MockMetaDataDao) GetById(id *int64) (*m.MetaData, error) {
	for _, i := range a.MetaDatas {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("metadata")
}

func (a *MockMetaDataDao) Create(src *m.MetaData) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockMetaDataDao) Update(src *m.MetaData) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockMetaDataDao) Delete(id *int64) error {
	panic("not implemented") // TODO: Implement
}

func (md *MockMetaDataDao) GetSuperKeySteps(_ int64) ([]m.MetaData, error) {
	panic("not implemented")
}

func (md *MockMetaDataDao) GetSuperKeyAccountNumber(applicationTypeId int64) (string, error) {
	panic("not implemented!")
}

func (a *MockApplicationTypeDao) Create(src *m.ApplicationType) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationTypeDao) Update(src *m.ApplicationType) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationTypeDao) Delete(id *int64) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationTypeDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	var appTypesOut []m.ApplicationType

	switch object := primaryCollection.(type) {
	case m.Source:
		var sourceExists bool
		for _, src := range fixtures.TestSourceData {
			if src.ID == object.ID {
				sourceExists = true
			}
		}

		if !sourceExists {
			return nil, 0, util.NewErrNotFound("source")
		}

		var appTypeIdList []int
		for _, app := range fixtures.TestApplicationData {
			if app.SourceID == object.ID {
				appTypeIdList = append(appTypeIdList, int(app.ApplicationTypeID))
			}
		}

		for _, appType := range a.ApplicationTypes {
			for _, id := range appTypeIdList {
				if appType.Id == int64(id) {
					appTypesOut = append(appTypesOut, appType)
					break
				}
			}
		}

	default:
		return nil, 0, fmt.Errorf("unexpected primary collection type")
	}

	count := int64(len(appTypesOut))

	return appTypesOut, count, nil
}

func (a *MockApplicationTypeDao) ApplicationTypeCompatibleWithSource(_, _ int64) error {
	if a.Compatible {
		return nil
	}

	return errors.New("Not compatible!")
}

func (at *MockApplicationTypeDao) GetSuperKeyResultType(applicationTypeId int64, authType string) (string, error) {
	panic("not needed")
}
func (a *MockApplicationTypeDao) ApplicationTypeCompatibleWithSourceType(_, _ int64) error {
	if a.Compatible {
		return nil
	}

	return errors.New("Not compatible!")
}

func (a *MockApplicationTypeDao) GetByName(_ string) (*m.ApplicationType, error) {
	return nil, nil
}

func (a *MockSourceTypeDao) List(limit int, offset int, filters []util.Filter) ([]m.SourceType, int64, error) {
	count := int64(len(a.SourceTypes))
	return a.SourceTypes, count, nil
}

func (a *MockSourceTypeDao) GetById(id *int64) (*m.SourceType, error) {
	for _, i := range a.SourceTypes {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source type")
}

func (a *MockSourceTypeDao) GetByName(_ string) (*m.SourceType, error) {
	return nil, nil
}

func (a *MockSourceTypeDao) Create(src *m.SourceType) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockSourceTypeDao) Update(src *m.SourceType) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockSourceTypeDao) Delete(id *int64) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	var applications []m.Application

	for index, i := range a.Applications {
		switch object := primaryCollection.(type) {
		case m.Source:
			if i.SourceID == object.ID {
				applications = append(applications, a.Applications[index])
			}
		default:
			return nil, 0, fmt.Errorf("unexpected primary collection type")
		}
	}
	count := int64(len(applications))
	if count == 0 {
		return nil, count, util.NewErrNotFound("application")
	}

	return applications, count, nil
}

func (a *MockApplicationDao) List(limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	count := int64(len(a.Applications))
	return a.Applications, count, nil
}

func (a *MockApplicationDao) GetById(id *int64) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("application")
}

func (a *MockApplicationDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("application")
}

func (a *MockApplicationDao) Create(src *m.Application) error {
	return nil
}

func (a *MockApplicationDao) Update(src *m.Application) error {
	return nil
}

func (a *MockApplicationDao) Delete(id *int64) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}
	return nil, util.NewErrNotFound("application")
}

func (a *MockApplicationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (m *MockApplicationDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockApplicationDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) error {
	return nil
}

func (m *MockApplicationDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (a *MockApplicationDao) Pause(_ int64) error {
	return nil
}

func (a *MockApplicationDao) Unpause(_ int64) error {
	return nil
}

func (src *MockApplicationDao) IsSuperkey(id int64) bool {
	return false
}

func (a *MockEndpointDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	var endpoints []m.Endpoint

	for index, i := range a.Endpoints {
		switch object := primaryCollection.(type) {
		case m.Source:
			if i.SourceID == object.ID {
				endpoints = append(endpoints, a.Endpoints[index])
			}
		default:
			return nil, 0, fmt.Errorf("unexpected primary collection type")
		}
	}
	count := int64(len(endpoints))
	if count == 0 {
		return nil, count, util.NewErrNotFound("endpoint")
	}

	return endpoints, count, nil
}

func (a *MockEndpointDao) List(limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	count := int64(len(a.Endpoints))
	return a.Endpoints, count, nil
}

func (a *MockEndpointDao) GetById(id *int64) (*m.Endpoint, error) {
	for _, app := range a.Endpoints {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("endpoint")
}

func (a *MockEndpointDao) Create(src *m.Endpoint) error {
	return nil
}

func (a *MockEndpointDao) Update(endpoint *m.Endpoint) error {
	if endpoint.ID == fixtures.TestEndpointData[0].ID {
		return nil
	}

	return util.NewErrNotFound("endpoint")
}

func (a *MockEndpointDao) Delete(id *int64) (*m.Endpoint, error) {
	panic("not implemented") // TODO: Implement
}

func (m *MockEndpointDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (m *MockEndpointDao) CanEndpointBeSetAsDefaultForSource(sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) IsRoleUniqueForSource(role string, sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) SourceHasEndpoints(sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockEndpointDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) error {
	return nil
}

func (m *MockEndpointDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (m *MockRhcConnectionDao) List(limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error) {
	count := int64(len(m.RhcConnections))
	return m.RhcConnections, count, nil
}

func (mr *MockRhcConnectionDao) GetById(id *int64) (*m.RhcConnection, error) {
	// The ".ToResponse" method of the RhcConnection expects to have at least one related source.
	source := []m.Source{
		{
			ID: 1,
		},
	}

	for _, rhcConnection := range mr.RhcConnections {
		if rhcConnection.ID == *id {
			rhcConnection.Sources = source
			return &rhcConnection, nil
		}
	}

	return nil, util.NewErrNotFound("rhcConnection")
}

func (mr *MockRhcConnectionDao) Create(rhcConnection *m.RhcConnection) (*m.RhcConnection, error) {
	// The ".ToResponse" method of the RhcConnection expects to have at least one related source.
	source := []m.Source{
		{
			ID: 1,
		},
	}

	rhcConnection.Sources = source
	return rhcConnection, nil
}

func (m *MockRhcConnectionDao) Update(rhcConnection *m.RhcConnection) error {
	for _, rhcTmp := range m.RhcConnections {
		if rhcTmp.ID == rhcConnection.ID {
			return nil
		}
	}

	return util.NewErrNotFound("rhcConnection")
}

func (m *MockRhcConnectionDao) Delete(id *int64) (*m.RhcConnection, error) {
	for _, rhcTmp := range m.RhcConnections {
		if rhcTmp.ID == *id {
			return &rhcTmp, nil
		}
	}

	return nil, util.NewErrNotFound("rhcConnection")
}

func (m *MockRhcConnectionDao) ListForSource(sourceId *int64, limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error) {
	count := int64(len(m.RelatedRhcConnections))

	return m.RelatedRhcConnections, count, nil
}

func (m MockApplicationAuthenticationDao) List(limit, offset int, filters []util.Filter) ([]m.ApplicationAuthentication, int64, error) {
	count := int64(len(m.ApplicationAuthentications))
	return m.ApplicationAuthentications, count, nil
}

func (m MockApplicationAuthenticationDao) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	for _, appAuth := range m.ApplicationAuthentications {
		if appAuth.ID == *id {
			return &appAuth, nil
		}
	}

	return nil, util.NewErrNotFound("application authentication")
}

func (m MockApplicationAuthenticationDao) Create(src *m.ApplicationAuthentication) error {
	return nil
}

func (m MockApplicationAuthenticationDao) Update(src *m.ApplicationAuthentication) error {
	panic("implement me")
}

func (m MockApplicationAuthenticationDao) Delete(id *int64) (*m.ApplicationAuthentication, error) {
	return m.GetById(id)
}

func (m MockApplicationAuthenticationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (m MockApplicationAuthenticationDao) ApplicationAuthenticationsByResource(_ string, _ []m.Application, _ []m.Authentication) ([]m.ApplicationAuthentication, error) {
	return m.ApplicationAuthentications, nil
}

func (m MockAuthenticationDao) List(limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	count := int64(len(m.Authentications))
	return m.Authentications, count, nil
}

func (m MockAuthenticationDao) GetById(id string) (*m.Authentication, error) {
	for _, auth := range m.Authentications {
		if auth.ID == id {
			return &auth, nil
		}
	}

	return nil, util.NewErrNotFound("authentication")
}

func (mAuth MockAuthenticationDao) ListForSource(sourceID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	sourceExists := false

	for _, src := range fixtures.TestSourceData {
		if src.ID == sourceID {
			sourceExists = true
		}
	}

	if !sourceExists {
		return nil, 0, util.NewErrNotFound("source")
	}

	count := int64(len(mAuth.Authentications))

	out := make([]m.Authentication, 0)

	for _, auth := range mAuth.Authentications {
		if auth.SourceID == sourceID {
			out = append(out, auth)
		}
	}

	return out, count, nil
}

func (m MockAuthenticationDao) ListForApplication(applicationID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) ListForApplicationAuthentication(appAuthID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) Create(auth *m.Authentication) error {
	switch auth.ResourceType {
	case "Application", "Endpoint", "Source":
		return nil
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}
}

func (m MockAuthenticationDao) Update(auth *m.Authentication) error {
	if auth.ID == fixtures.TestAuthenticationData[0].ID {
		return nil
	}
	return util.NewErrNotFound("authentication")
}

func (m MockAuthenticationDao) Delete(id string) (*m.Authentication, error) {
	for _, auth := range m.Authentications {
		if auth.ID == id {
			return &auth, nil
		}
	}
	return nil, util.NewErrNotFound("authentication")
}

func (m MockAuthenticationDao) Tenant() *int64 {
	panic("implement me")
}

func (m MockAuthenticationDao) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	panic("implement me")
}

func (m MockAuthenticationDao) ToEventJSON(resource util.Resource) ([]byte, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkCreate(auth *m.Authentication) error {
	panic("AAA")
}
