package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type SourceTypeDaoImpl struct {
}

func (st *SourceTypeDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.SourceType, int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sourceTypes := make([]m.SourceType, 0, limit)
	query := DB.Debug().Model(&m.SourceType{})

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sourceTypes)

	return sourceTypes, count, result.Error
}

func (st *SourceTypeDaoImpl) GetById(id *int64) (*m.SourceType, error) {
	sourceType := &m.SourceType{Id: *id}
	result := DB.Debug().First(sourceType)
	if result.Error != nil {
		return nil, util.NewErrNotFound("source type")
	}
	return sourceType, nil
}

func (st *SourceTypeDaoImpl) Create(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (st *SourceTypeDaoImpl) Update(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (st *SourceTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}
