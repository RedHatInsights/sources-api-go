package dao

import (
	"encoding/json"
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type SourceDaoImpl struct {
	TenantID *int64
}

func (s *SourceDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sources := make([]m.Source, 0, limit)

	sourceType, err := m.NewRelationObject(primaryCollection, *s.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound(sourceType.StringBaseObject())
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

func (s *SourceDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
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

func (s *SourceDaoImpl) ListInternal(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	query := DB.Debug().
		Model(&m.Source{}).
		Joins("Tenant").
		Select(`sources.id, sources.availability_status, "Tenant".external_tenant`)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	sources := make([]m.Source, 0, limit)
	result := query.Offset(offset).Limit(limit).Find(&sources)

	// Getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	return sources, count, result.Error
}

func (s *SourceDaoImpl) GetById(id *int64) (*m.Source, error) {
	src := &m.Source{ID: *id}
	result := DB.First(src)
	if result.Error != nil {
		return nil, util.NewErrNotFound("source")
	}

	return src, nil
}

// Function that searches for a source and preloads any specified relations
func (s *SourceDaoImpl) GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error) {
	src := &m.Source{ID: *id}
	q := DB.Where("tenant_id = ?", s.TenantID)

	for _, preload := range preloads {
		q = q.Preload(preload)
	}

	result := q.First(&src)
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

func (s *SourceDaoImpl) Delete(id *int64) (*m.Source, error) {
	src := &m.Source{ID: *id}
	result := DB.Where("tenant_id = ?", s.TenantID).First(src)
	if result.Error != nil {
		return nil, util.NewErrNotFound("source")
	}

	if result := DB.Delete(src); result.Error != nil {
		return nil, fmt.Errorf("failed to delete source id %v", *id)
	}

	return src, nil
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

func (s *SourceDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	src := m.Source{ID: resource.ResourceID}
	result := DB.Find(&src)
	if result.Error != nil {
		return nil, result.Error
	}

	authentication := &m.Authentication{ResourceID: src.ID, ResourceType: "Source"}
	return BulkMessageFromSource(&src, authentication)
}

func (s *SourceDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	result := DB.Model(&m.Source{ID: resource.ResourceID}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("source not found %v", resource)
	}

	return nil
}

func (s *SourceDaoImpl) FindWithTenant(id *int64) (*m.Source, error) {
	src := &m.Source{ID: *id}
	result := DB.Preload("Tenant").Find(&src)

	return src, result.Error
}

func (s *SourceDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	src, err := s.FindWithTenant(&resource.ResourceID)
	if err != nil {
		return nil, err
	}

	data, errorJson := json.Marshal(src.ToEvent())
	if errorJson != nil {
		return nil, errorJson
	}

	return data, err
}
