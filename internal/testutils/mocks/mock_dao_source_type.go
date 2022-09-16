package mocks

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockSourceTypeDao struct {
	SourceTypes []m.SourceType
}

func (mockSourceTypeDao *MockSourceTypeDao) List(_ int, _ int, _ []util.Filter) ([]m.SourceType, int64, error) {
	count := int64(len(mockSourceTypeDao.SourceTypes))
	return mockSourceTypeDao.SourceTypes, count, nil
}

func (mockSourceTypeDao *MockSourceTypeDao) GetById(id *int64) (*m.SourceType, error) {
	for _, i := range mockSourceTypeDao.SourceTypes {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source type")
}

func (mockSourceTypeDao *MockSourceTypeDao) GetByName(_ string) (*m.SourceType, error) {
	return nil, nil
}

func (mockSourceTypeDao *MockSourceTypeDao) Create(_ *m.SourceType) error {
	panic("not implemented")
}

func (mockSourceTypeDao *MockSourceTypeDao) Update(_ *m.SourceType) error {
	panic("not implemented")
}

func (mockSourceTypeDao *MockSourceTypeDao) Delete(_ *int64) error {
	panic("not implemented")
}
