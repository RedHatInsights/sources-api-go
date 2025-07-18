package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// GetSourceTypeDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetSourceTypeDao func() SourceTypeDao

// getDefaultRhcConnectionDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultSourceTypeDao() SourceTypeDao {
	return &sourceTypeDaoImpl{}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetSourceTypeDao = getDefaultSourceTypeDao
}

type sourceTypeDaoImpl struct {
}

func (st *sourceTypeDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.SourceType, int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sourceTypes := make([]m.SourceType, 0, limit)
	query := DB.Debug().Model(&m.SourceType{})

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sourceTypes)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return sourceTypes, count, nil
}

func (st *sourceTypeDaoImpl) GetById(id *int64) (*m.SourceType, error) {
	var sourceType m.SourceType

	err := DB.Debug().
		Model(&m.SourceType{}).
		Where("id = ?", *id).
		First(&sourceType).
		Error
	if err != nil {
		return nil, util.NewErrNotFound("source type")
	}

	return &sourceType, nil
}

func (st *sourceTypeDaoImpl) GetByName(name string) (*m.SourceType, error) {
	sourceTypes := make([]m.SourceType, 0)
	result := DB.Debug().
		Where("name LIKE ?", "%"+name+"%").
		Find(&sourceTypes)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected > int64(1) {
		return nil, util.NewErrBadRequest("Found more than one of the same source type name")
	} else if result.RowsAffected == int64(0) {
		return nil, util.NewErrNotFound("source type")
	}

	return &sourceTypes[0], nil
}

func (st *sourceTypeDaoImpl) Create(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (st *sourceTypeDaoImpl) Update(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (st *sourceTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}
