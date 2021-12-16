package dao

import (
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type SourceDaoImpl struct {
	TenantID *int64
}

func (s *SourceDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []middleware.Filter) ([]m.Source, int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sources := make([]m.Source, 0, limit)

	sourceType, err := m.NewRelationObject(primaryCollection, *s.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
	}
	query := sourceType.HasMany(&m.Source{}, DB.Debug())

	query = query.Where("sources.tenant_id = ?", s.TenantID)

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.Source{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sources)

	return sources, count, result.Error
}

func (s *SourceDaoImpl) List(limit, offset int, filters []middleware.Filter) ([]m.Source, int64, error) {
	sources := make([]m.Source, 0, limit)
	query := DB.Debug().Model(&m.Source{}).
		Offset(offset).
		Where("tenant_id = ?", s.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Find(&sources)

	return sources, count, result.Error
}

func (s *SourceDaoImpl) GetById(id *int64) (*m.Source, error) {
	src := &m.Source{ID: *id}
	result := DB.First(src)

	return src, result.Error
}

func (s *SourceDaoImpl) Create(src *m.Source) error {
	src.TenantID = *s.TenantID // the TenantID gets injected in the middleware
	result := DB.Create(src)
	return result.Error
}

func (s *SourceDaoImpl) Update(src *m.Source) error {
	result := DB.Updates(src)
	return result.Error
}

func (s *SourceDaoImpl) Delete(id *int64) error {
	src := &m.Source{ID: *id}
	if result := DB.Delete(src); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete source id %v", *id)
	}

	return nil
}

func (s *SourceDaoImpl) Tenant() *int64 {
	return s.TenantID
}

func (s *SourceDaoImpl) NameExistsInCurrentTenant(name string) bool {
	src := &m.Source{Name: name}
	result := DB.Where("name = ? AND tenant_id = ?", name, s.TenantID).First(src)

	// If the name is found, GORM returns one row and no errors.
	return result.Error == nil
}

func (s *SourceDaoImpl) BulkMessage(id *int64) (map[string]interface{}, error) {
	src := m.Source{ID: *id}
	result := DB.Find(&src)
	if result.Error != nil {
		return nil, result.Error
	}

	return BulkMessageFrom(&src)
}

func (s *SourceDaoImpl) FetchAndUpdateBy(id *int64, updateAttributes map[string]interface{}) error {
	result := DB.Model(&m.Source{ID: *id}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("source not found %v", id)
	}

	return nil
}

func (s *SourceDaoImpl) FindWithTenant(id *int64) (*m.Source, error) {
	src := &m.Source{ID: *id}
	result := DB.Preload("Tenant").Find(&src)

	return src, result.Error
}

func (s *SourceDaoImpl) ToEventJSON(id *int64) ([]byte, error) {
	src, err := s.FindWithTenant(id)
	if err != nil {
		return nil, err
	}

	data, errorJson := json.Marshal(src.ToEvent())
	if errorJson != nil {
		return nil, errorJson
	}

	return data, err
}
