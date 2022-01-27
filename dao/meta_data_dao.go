package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MetaDataDaoImpl struct {
	TenantID *int64
}

func (a *MetaDataDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	metadatas := make([]m.MetaData, 0, limit)
	collection, err := m.NewRelationObject(primaryCollection, -1, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("application type")
	}

	query := collection.HasMany(&m.MetaData{}, DB.Debug())
	query = query.Where("meta_data.type = 'AppMetaData'")

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Model(&m.MetaData{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&metadatas)
	return metadatas, count, result.Error
}

func (a *MetaDataDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	metaData := make([]m.MetaData, 0, limit)
	query := DB.Debug().Model(&m.MetaData{})

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&metaData)
	return metaData, count, result.Error
}

func (a *MetaDataDaoImpl) GetById(id *int64) (*m.MetaData, error) {
	metaData := &m.MetaData{ID: *id}
	result := DB.First(&metaData)
	if result.Error != nil {
		return nil, util.NewErrNotFound("metadata")
	}

	return metaData, nil
}

func (a *MetaDataDaoImpl) Create(metaData *m.MetaData) error {
	result := DB.Create(metaData)
	return result.Error
}

func (a *MetaDataDaoImpl) Update(metaData *m.MetaData) error {
	result := DB.Updates(metaData)
	return result.Error
}

func (a *MetaDataDaoImpl) Delete(id *int64) error {
	metaData := &m.MetaData{ID: *id}
	if result := DB.Delete(metaData); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete application id %v", *id)
	}

	return nil
}
