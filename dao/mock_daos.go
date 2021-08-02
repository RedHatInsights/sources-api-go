package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type MockSourceDao struct {
	Sources []m.Source
}

type MockApplicationDao struct {
	Applications []m.Application
}

type MockApplicationTypeDao struct {
	ApplicationTypes []m.ApplicationType
}

type MockSourceTypeDao struct {
	SourceTypes []m.SourceType
}

func (m *MockSourceDao) List(limit, offset int, filters []middleware.Filter) ([]m.Source, *int64, error) {
	count := int64(len(m.Sources))
	return m.Sources, &count, nil
}

func (m *MockSourceDao) GetById(id *int64) (*m.Source, error) {
	for _, i := range m.Sources {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("source not found")
}

func (m *MockSourceDao) Create(src *m.Source) error {
	return nil
}

func (m *MockSourceDao) Update(src *m.Source) error {
	panic("implement me")
}

func (m *MockSourceDao) Delete(id *int64) error {
	panic("implement me")
}

func (m *MockSourceDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (a *MockApplicationTypeDao) List(limit int, offset int, filters []middleware.Filter) ([]m.ApplicationType, *int64, error) {
	count := int64(len(a.ApplicationTypes))
	return a.ApplicationTypes, &count, nil
}

func (a *MockApplicationTypeDao) GetById(id *int64) (*m.ApplicationType, error) {
	for _, i := range a.ApplicationTypes {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("application Type not found")
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

func (a *MockSourceTypeDao) List(limit int, offset int, filters []middleware.Filter) ([]m.SourceType, *int64, error) {
	count := int64(len(a.SourceTypes))
	return a.SourceTypes, &count, nil
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

func (a *MockApplicationDao) List(limit int, offset int, filters []middleware.Filter) ([]m.Application, *int64, error) {
	count := int64(len(a.Applications))
	return a.Applications, &count, nil
}

func (a *MockApplicationDao) GetById(id *int64) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.Id == *id {
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
