package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type MockSourceDao struct {
	Sources []m.Source
}

type MockApplicationTypeDao struct {
	ApplicationTypes []m.ApplicationType
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
