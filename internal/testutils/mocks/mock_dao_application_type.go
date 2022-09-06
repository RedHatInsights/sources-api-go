package mocks

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockApplicationTypeDao struct {
	ApplicationTypes []m.ApplicationType
	Compatible       bool
}

func (mockAppTypeDao *MockApplicationTypeDao) List(_ int, _ int, _ []util.Filter) ([]m.ApplicationType, int64, error) {
	count := int64(len(mockAppTypeDao.ApplicationTypes))
	return mockAppTypeDao.ApplicationTypes, count, nil
}

func (mockAppTypeDao *MockApplicationTypeDao) GetById(id *int64) (*m.ApplicationType, error) {
	for _, i := range mockAppTypeDao.ApplicationTypes {
		if i.Id == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("application type")
}

func (mockAppTypeDao *MockApplicationTypeDao) Create(_ *m.ApplicationType) error {
	panic("not implemented")
}

func (mockAppTypeDao *MockApplicationTypeDao) Update(_ *m.ApplicationType) error {
	panic("not implemented")
}

func (mockAppTypeDao *MockApplicationTypeDao) Delete(_ *int64) error {
	panic("not implemented")
}

func (mockAppTypeDao *MockApplicationTypeDao) SubCollectionList(primaryCollection interface{}, _, _ int, _ []util.Filter) ([]m.ApplicationType, int64, error) {
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

		for _, appType := range mockAppTypeDao.ApplicationTypes {
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

func (mockAppTypeDao *MockApplicationTypeDao) ApplicationTypeCompatibleWithSource(_, _ int64) error {
	if mockAppTypeDao.Compatible {
		return nil
	}

	return errors.New("Not compatible!")
}

func (mockAppTypeDao *MockApplicationTypeDao) GetSuperKeyResultType(_ int64, _ string) (string, error) {
	panic("not needed")
}

func (mockAppTypeDao *MockApplicationTypeDao) ApplicationTypeCompatibleWithSourceType(_, _ int64) error {
	if mockAppTypeDao.Compatible {
		return nil
	}

	return errors.New("Not compatible!")
}

func (mockAppTypeDao *MockApplicationTypeDao) GetByName(_ string) (*m.ApplicationType, error) {
	return nil, nil
}
