package mocks

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockMetaDataDao struct {
	MetaDatas []m.MetaData
}

func (mockMetaDataDao *MockMetaDataDao) List(_ int, _ int, _ []util.Filter) ([]m.MetaData, int64, error) {
	count := int64(len(mockMetaDataDao.MetaDatas))
	return mockMetaDataDao.MetaDatas, count, nil
}

func (mockMetaDataDao *MockMetaDataDao) SubCollectionList(primaryCollection interface{}, _, _ int, _ []util.Filter) ([]m.MetaData, int64, error) {
	var appMetaDataList []m.MetaData

	for index, i := range mockMetaDataDao.MetaDatas {
		switch object := primaryCollection.(type) {
		case m.ApplicationType:
			if i.ApplicationTypeID == object.Id {
				appMetaDataList = append(appMetaDataList, mockMetaDataDao.MetaDatas[index])
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

func (mockMetaDataDao *MockMetaDataDao) GetById(id *int64) (*m.MetaData, error) {
	for _, i := range mockMetaDataDao.MetaDatas {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("metadata")
}

func (mockMetaDataDao *MockMetaDataDao) Create(_ *m.MetaData) error {
	panic("not implemented")
}

func (mockMetaDataDao *MockMetaDataDao) Update(_ *m.MetaData) error {
	panic("not implemented")
}

func (mockMetaDataDao *MockMetaDataDao) Delete(_ *int64) error {
	panic("not implemented")
}

func (mockMetaDataDao *MockMetaDataDao) GetSuperKeySteps(_ int64) ([]m.MetaData, error) {
	panic("not implemented")
}

func (mockMetaDataDao *MockMetaDataDao) GetSuperKeyAccountNumber(_ int64) (string, error) {
	panic("not implemented!")
}

func (mockMetaDataDao *MockMetaDataDao) ApplicationOptedIntoRetry(_ int64) (bool, error) {
	panic("not implemented!")
}
