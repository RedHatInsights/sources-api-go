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
