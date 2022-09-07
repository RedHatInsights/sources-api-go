package mocks

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockSourceTypeDao struct {
	SourceTypes []m.SourceType
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
