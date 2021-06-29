package dao

import (
	"fmt"

	"github.com/lindgrenj6/sources-api-go/middleware"
	m "github.com/lindgrenj6/sources-api-go/model"
)

type MockSourceDao struct {
	Sources []m.Source
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
