package dao

import (
	"strings"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const (
	AWS_WIZARD_ACCOUNT_NUMBER_SETTING = "aws_wizard_account_number"
	RETRY_SOURCE_CREATION_SETTING     = "retry_source_creation"
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
	relationObject, err := m.NewRelationObject(primaryCollection, -1, DB.Debug())
	if err != nil {
		return nil, 0, err
	}

	query := relationObject.HasMany(&m.MetaData{}, DB.Debug())
	query = query.Where("meta_data.type = ?", m.APP_META_DATA)

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
	query := DB.Debug().Model(&m.MetaData{}).Where("type = ?", m.APP_META_DATA)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&metaData)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return metaData, count, nil
}

func (md *metaDataDaoImpl) GetById(id *int64) (*m.MetaData, error) {
	var metaData m.MetaData

	err := DB.
		Debug().
		Model(&m.MetaData{}).
		Where("id = ?", *id).
		First(&metaData).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("metadata")
	}

	return &metaData, nil
}

func (md *metaDataDaoImpl) GetSuperKeySteps(applicationTypeId int64) ([]m.MetaData, error) {
	steps := make([]m.MetaData, 0)

	result := DB.Model(&m.MetaData{}).
		Where("type = ?", m.SUPERKEY_META_DATA).
		Where("application_type_id = ?", applicationTypeId).
		Order("step").
		Scan(&steps)

	return steps, result.Error
}

func (md *metaDataDaoImpl) GetSuperKeyAccountNumber(applicationTypeId int64) (string, error) {
	var account string
	result := DB.Model(&m.MetaData{}).
		Select("payload").
		Where("type = ?", m.APP_META_DATA).
		Where("application_type_id = ?", applicationTypeId).
		Where("name = ?", AWS_WIZARD_ACCOUNT_NUMBER_SETTING).
		First(&account)

	// it gets stored as `"12345"` but we do not want the quotes - remove them here.
	return strings.ReplaceAll(account, `"`, ``), result.Error
}

func (md *metaDataDaoImpl) ApplicationOptedIntoRetry(applicationTypeId int64) (bool, error) {
	var optIn bool

	result := DB.Debug().
		Model(&m.MetaData{}).
		Select(`payload::text = '"true"'`).
		Where("name = ?", RETRY_SOURCE_CREATION_SETTING).
		Where("type = ?", m.APP_META_DATA).
		Where("application_type_id = ?", applicationTypeId).
		Scan(&optIn)

	return optIn, result.Error
}
