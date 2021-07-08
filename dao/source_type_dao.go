package dao

import (
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type SourceTypeDaoImpl struct {
}

func (a *SourceTypeDaoImpl) List(limit, offset int, filters []middleware.Filter) ([]m.SourceType, *int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sourceTypes := make([]m.SourceType, 0, limit)
	query := DB.Debug()

	err := applyFilters(query, filters)
	if err != nil {
		return nil, nil, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.SourceType{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sourceTypes)

	return sourceTypes, &count, result.Error
}

func (a *SourceTypeDaoImpl) GetById(id *int64) (*m.SourceType, error) {
	sourceType := &m.SourceType{Id: *id}
	result := DB.Debug().First(sourceType)

	return sourceType, result.Error
}

func (a *SourceTypeDaoImpl) Create(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *SourceTypeDaoImpl) Update(_ *m.SourceType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *SourceTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}
