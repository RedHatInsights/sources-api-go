package dao

import (
	"fmt"

	"github.com/lindgrenj6/sources-api-go/middleware"
	m "github.com/lindgrenj6/sources-api-go/model"
)

type SourceDaoImpl struct {
	TenantID *int64
}

func (s *SourceDaoImpl) List(limit, offset int, filters []middleware.Filter) ([]m.Source, *int64, error) {
	sources := make([]m.Source, 0, limit)
	query := DB.Debug().
		Offset(offset).
		Where("tenant_id = ?", s.TenantID)

	err := applyFilters(query, filters)
	if err != nil {
		return nil, nil, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.Source{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Find(&sources)

	return sources, &count, result.Error
}

func (s *SourceDaoImpl) GetById(id *int64) (*m.Source, error) {
	src := &m.Source{Id: *id}
	result := DB.First(src)

	return src, result.Error
}

func (s *SourceDaoImpl) Create(src *m.Source) error {
	result := DB.Create(src)
	return result.Error
}

func (s *SourceDaoImpl) Update(src *m.Source) error {
	result := DB.Updates(src)
	return result.Error
}

func (s *SourceDaoImpl) Delete(id *int64) error {
	src := &m.Source{Id: *id}
	if result := DB.Delete(src); result.RowsAffected == 0 {
		return fmt.Errorf("faile to delete source id %v", *id)
	}

	return nil
}

func (s *SourceDaoImpl) Tenant() *int64 {
	return s.TenantID
}
