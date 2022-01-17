package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockSourceDao struct {
	Sources []m.Source
}

type MockApplicationDao struct {
	Applications []m.Application
}

type MockEndpointDao struct {
	Endpoints []m.Endpoint
}
type MockApplicationTypeDao struct {
	ApplicationTypes []m.ApplicationType
}

type MockSourceTypeDao struct {
	SourceTypes []m.SourceType
}

type MockMetaDataDao struct {
	MetaDatas []m.MetaData
}

func (src *MockSourceDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(1)

	var sources []m.Source

	for index, i := range src.Sources {
		switch object := primaryCollection.(type) {
		case m.SourceType:
			if i.SourceTypeID == object.Id {
				sources = append(sources, src.Sources[index])
			}
		case m.ApplicationType:
			if i.ID == 1 { // Source with ID=1 is relevant for application types/:id/sources test
				sources = append(sources, src.Sources[index])
			}
		}

	}

	return sources, count, nil
}

func (src *MockSourceDao) List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(src.Sources))
	return src.Sources, count, nil
}

func (src *MockSourceDao) GetById(id *int64) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("source not found")
}

func (src *MockSourceDao) Create(s *m.Source) error {
	return nil
}

func (src *MockSourceDao) Update(s *m.Source) error {
	panic("implement me")
}

func (src *MockSourceDao) Delete(id *int64) error {
	panic("implement me")
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

func (src *MockSourceDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("source not found")
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

	return nil, fmt.Errorf("application Type not found")
}

func (a *MockMetaDataDao) List(limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	count := int64(len(a.MetaDatas))
	return a.MetaDatas, count, nil
}

func (a *MockMetaDataDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	count := int64(len(a.MetaDatas))
	return a.MetaDatas, count, nil
}

func (a *MockMetaDataDao) GetById(id *int64) (*m.MetaData, error) {
	for _, i := range a.MetaDatas {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("application Type not found")
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
	count := int64(1) // ApplicationType ID=1

	return []m.ApplicationType{a.ApplicationTypes[0]}, count, nil
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

	return nil, fmt.Errorf("application Type not found")
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
	count := int64(len(a.Applications))
	return a.Applications, count, nil
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

	return nil, fmt.Errorf("application not found")
}

func (a *MockApplicationDao) Create(src *m.Application) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationDao) Update(src *m.Application) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationDao) Delete(id *int64) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockApplicationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (a *MockEndpointDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	count := int64(len(a.Endpoints))
	return a.Endpoints, count, nil
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

	return nil, fmt.Errorf("endpoint not found")
}

func (a *MockEndpointDao) Create(src *m.Endpoint) error {
	return nil
}

func (a *MockEndpointDao) Update(src *m.Endpoint) error {
	panic("not implemented") // TODO: Implement
}

func (a *MockEndpointDao) Delete(id *int64) error {
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
