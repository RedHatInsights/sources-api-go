package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// GetMetaDataDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetMetaDataDao func() MetaDataDao

// getDefaultMetaDataDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultMetaDataDao() MetaDataDao {
	return &metaDataDaoImpl{}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetMetaDataDao = getDefaultMetaDataDao
}

type metaDataDaoImpl struct{}

func (md *metaDataDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	metadatas := make([]m.MetaData, 0, limit)
	collection, err := m.NewRelationObject(primaryCollection, -1, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("application type")
	}

	query := collection.HasMany(&m.MetaData{}, DB.Debug())
	query = query.Where("meta_data.type = 'AppMetaData'")

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Model(&m.MetaData{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&metadatas)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return metadatas, count, result.Error
}

func (md *metaDataDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.MetaData, int64, error) {
	metaData := make([]m.MetaData, 0, limit)
	query := DB.Debug().Model(&m.MetaData{}).Where("type = 'AppMetaData'")

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&metaData)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return metaData, count, nil
}

func (md *metaDataDaoImpl) GetById(id *int64) (*m.MetaData, error) {
	metaData := &m.MetaData{ID: *id}
	result := DB.First(&metaData)
	if result.Error != nil {
		return nil, util.NewErrNotFound("metadata")
	}

	return metaData, nil
}

func (md *metaDataDaoImpl) GetSuperKeySteps(applicationTypeId int64) ([]m.MetaData, error) {
	steps := make([]m.MetaData, 0)

	result := DB.Model(&m.MetaData{}).
		Where("type = 'SuperKeyMetaData'").
		Where("application_type_id = ?", applicationTypeId).
		Order("step").
		Scan(&steps)

	return steps, result.Error
}
