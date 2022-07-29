package dao

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
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

		// Application type exists = find related sources
		sources = testutils.GetSourcesWithAppType(object.Id)

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
func (src *MockSourceDao) User() *int64 {
	user := int64(1)
	return &user
}

func (src *MockSourceDao) Exists(sourceId int64) (bool, error) {
	for _, source := range src.Sources {
		if source.ID == sourceId {
			return true, nil
		}
	}

	return false, nil
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

func (msd *MockSourceDao) DeleteCascade(id int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error) {
	var source *m.Source
	for _, src := range fixtures.TestSourceData {
		if src.ID == id {
			source = &src
		}
	}

	if source == nil {
		return nil, nil, nil, nil, nil, util.NewErrNotFound("source")
	}

	return fixtures.TestApplicationAuthenticationData, fixtures.TestApplicationData, fixtures.TestEndpointData, fixtures.TestRhcConnectionData, source, nil
}

func (m *MockSourceDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSourceDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
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

func (md *MockMetaDataDao) ApplicationOptedIntoRetry(applicationTypeId int64) (bool, error) {
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

		appTypes := make(map[int64]int)
		for _, app := range fixtures.TestApplicationData {
			if app.SourceID == object.ID {
				appTypes[app.ApplicationTypeID]++
			}
		}

		for _, appType := range a.ApplicationTypes {
			for id := range appTypes {
				if appType.Id == id {
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

	switch object := primaryCollection.(type) {
	case m.Source:
		var sourceExists bool

		for _, s := range fixtures.TestSourceData {
			if s.ID == object.ID {
				sourceExists = true
			}
		}
		// if source doesn't exist, return Not Found Err
		if !sourceExists {
			return nil, 0, util.NewErrNotFound("source")
		}

		// else return list of related applications
		for _, app := range a.Applications {
			if object.ID == app.SourceID {
				applications = append(applications, app)
			}
		}
	}

	return applications, int64(len(applications)), nil
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
			for _, preload := range preloads {
				if strings.Contains(strings.ToLower(preload), "source") {
					app.Source = fixtures.TestSourceData[0]
				}
			}

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

func (a *MockApplicationDao) User() *int64 {
	user := int64(1)
	return &user
}

func (a *MockApplicationDao) DeleteCascade(applicationId int64) ([]m.ApplicationAuthentication, *m.Application, error) {
	var application *m.Application
	for _, app := range fixtures.TestApplicationData {
		if app.ID == applicationId {
			application = &app
		}
	}

	if application == nil {
		return nil, nil, util.NewErrNotFound("application")
	}

	return fixtures.TestApplicationAuthenticationData, application, nil
}

func (a *MockApplicationDao) Exists(applicationId int64) (bool, error) {
	for _, application := range a.Applications {
		if application.ID == applicationId {
			return true, nil
		}
	}

	return false, nil
}

func (m *MockApplicationDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockApplicationDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
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
	// Return not found err when the source doesn't exist
	var sourceExist bool
	var sourceId int64
	switch object := primaryCollection.(type) {
	case m.Source:
		for _, source := range fixtures.TestSourceData {
			if object.ID == source.ID {
				sourceExist = true
				sourceId = object.ID
				break
			}
		}
	default:
		return nil, 0, fmt.Errorf("unexpected primary collection type")
	}

	if !sourceExist {
		return nil, 0, util.NewErrNotFound("source")
	}

	// Get list of endpoints for existing source
	var endpointsOut []m.Endpoint
	for _, e := range a.Endpoints {
		if e.SourceID == sourceId {
			endpointsOut = append(endpointsOut, e)
		}
	}

	return endpointsOut, int64(len(endpointsOut)), nil
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

func (endpointDao *MockEndpointDao) Delete(id *int64) (*m.Endpoint, error) {
	for i, e := range endpointDao.Endpoints {
		if e.ID == *id {
			endpointDao.Endpoints = append(endpointDao.Endpoints[:i], endpointDao.Endpoints[i+1:]...)

			return &e, nil
		}
	}
	return nil, util.NewErrNotFound("endpoint")
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

func (m *MockEndpointDao) Exists(endpointId int64) (bool, error) {
	return true, nil
}

func (m *MockEndpointDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockEndpointDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
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
	// Check if in fixtures is a source with given source id
	var sourceExists bool
	for _, src := range fixtures.TestSourceData {
		if src.ID == rhcConnection.Sources[0].ID {
			sourceExists = true
		}
	}

	if !sourceExists {
		return nil, util.NewErrNotFound("source")
	}

	// Check if in fixtures exists same relation
	var relationExists bool
	for _, s := range fixtures.TestSourceRhcConnectionData {
		if s.SourceId == rhcConnection.Sources[0].ID {
			for _, r := range fixtures.TestRhcConnectionData {
				if s.RhcConnectionId == r.ID && r.RhcId == rhcConnection.RhcId {
					relationExists = true
				}
			}
		}
	}

	if relationExists {
		return nil, util.NewErrBadRequest("connection already exists")
	}

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
		// If secret store is database, we compare given ID with different field
		// than if secret store is vault
		if conf.SecretStore == "database" {
			if fmt.Sprintf("%d", auth.DbID) == id {
				return &auth, nil
			}
		} else {
			if auth.ID == id {
				return &auth, nil
			}
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

	out := make([]m.Authentication, 0)

	for _, auth := range mAuth.Authentications {
		if auth.SourceID == sourceID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mAuth MockAuthenticationDao) ListForApplication(appId int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	var appExists bool
	for _, app := range fixtures.TestApplicationData {
		if app.ID == appId {
			appExists = true
		}
	}
	if !appExists {
		return nil, 0, util.NewErrNotFound("application")
	}

	var out []m.Authentication
	for _, auth := range fixtures.TestAuthenticationData {
		if auth.ResourceType == "Application" && auth.ResourceID == appId {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mAuthDao MockAuthenticationDao) ListForApplicationAuthentication(appAuthID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	var appAuthExists bool
	var authID int64

	for _, appAuth := range fixtures.TestApplicationAuthenticationData {
		if appAuth.ID == appAuthID {
			authID = appAuth.AuthenticationID
			appAuthExists = true
			break
		}
	}

	if !appAuthExists {
		return nil, 0, util.NewErrNotFound("application authentication")
	}

	auth, err := mAuthDao.GetById(fmt.Sprintf("%d", authID))
	if err != nil {
		return nil, 0, err
	}

	authentications := []m.Authentication{*auth}

	return authentications, int64(1), nil
}

func (mAuth MockAuthenticationDao) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	endpointExists := false

	for _, e := range fixtures.TestEndpointData {
		if e.ID == endpointID {
			endpointExists = true
			break
		}
	}

	if !endpointExists {
		return nil, 0, util.NewErrNotFound("endpoint")
	}

	out := make([]m.Authentication, 0)

	for _, auth := range mAuth.Authentications {
		if auth.ResourceType == "Endpoint" && auth.ResourceID == endpointID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
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
		// If secret store is database, we compare given ID with different field
		// than if secret store is vault
		if conf.SecretStore == "database" {
			if fmt.Sprintf("%d", auth.DbID) == id {
				return &auth, nil
			}
		} else {
			if auth.ID == id {
				return &auth, nil
			}
		}
	}
	return nil, util.NewErrNotFound("authentication")
}

func (m MockAuthenticationDao) Tenant() *int64 {
	fakeTenantId := int64(12345)
	return &fakeTenantId
}

func (m MockAuthenticationDao) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) ToEventJSON(resource util.Resource) ([]byte, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkCreate(auth *m.Authentication) error {
	panic("AAA")
}

func (mad MockAuthenticationDao) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
	var authsList []m.Authentication
	for _, auth := range fixtures.TestAuthenticationData {
		for _, rid := range resourceIds {
			if auth.ResourceType == resourceType && auth.ResourceID == rid {
				authsList = append(authsList, auth)
			}
		}
	}

	if authsList == nil {
		return nil, nil
	} else {
		return authsList, nil
	}
}

// BulkDelete deletes all the authentications given as a list, and returns the ones that were deleted.
func (m MockAuthenticationDao) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	return authentications, nil
}
