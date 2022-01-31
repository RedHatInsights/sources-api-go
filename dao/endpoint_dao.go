package dao

import (
	"encoding/json"
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type EndpointDaoImpl struct {
	TenantID *int64
}

func (a *EndpointDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	sourceType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	query := sourceType.HasMany(&m.Endpoint{}, DB.Debug())
	query = query.Where("endpoints.tenant_id = ?", a.TenantID)

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Model(&m.Endpoint{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&endpoints)
	return endpoints, count, result.Error
}

func (a *EndpointDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	query := DB.Debug().Model(&m.Endpoint{}).
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&endpoints)
	return endpoints, count, result.Error
}

func (a *EndpointDaoImpl) GetById(id *int64) (*m.Endpoint, error) {
	app := &m.Endpoint{ID: *id}
	result := DB.First(&app)
	if result.Error != nil {
		return nil, util.NewErrNotFound("endpoint")
	}

	return app, nil
}

func (a *EndpointDaoImpl) Create(app *m.Endpoint) error {
	app.TenantID = *a.TenantID

	result := DB.Create(app)
	return result.Error
}

func (a *EndpointDaoImpl) Update(app *m.Endpoint) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *EndpointDaoImpl) Delete(id *int64) (*m.Endpoint, error) {
	endpt := &m.Endpoint{ID: *id}
	result := DB.Where("tenant_id = ?", a.TenantID).First(&endpt)
	if result.Error != nil {
		return nil, util.NewErrNotFound("endpoint")
	}

	if result := DB.Delete(endpt); result.Error != nil {
		return nil, fmt.Errorf("failed to delete endpoint id %v", *id)
	}

	return endpt, nil
}

func (a *EndpointDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *EndpointDaoImpl) CanEndpointBeSetAsDefaultForSource(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	// add double quotes to the "default" column to avoid any clashes with postgres' "default" keyword
	result := DB.Where(`"default" = true AND source_id = ?`, sourceId).First(&endpoint)
	return result.Error != nil
}

func (a *EndpointDaoImpl) IsRoleUniqueForSource(role string, sourceId int64) bool {
	endpoint := &m.Endpoint{}
	result := DB.Where("role = ? AND source_id = ?", role, sourceId).First(&endpoint)

	// If the record doesn't exist "result.Error" will have a "record not found" error
	return result.Error != nil
}

func (a *EndpointDaoImpl) SourceHasEndpoints(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	result := DB.Where("source_id = ?", sourceId).First(&endpoint)

	return result.Error == nil
}

func (a *EndpointDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	endpoint := &m.Endpoint{ID: resource.ResourceID}
	result := DB.Preload("Source").Find(&endpoint)

	if result.Error != nil {
		return nil, result.Error
	}

	return BulkMessageFromSource(&endpoint.Source)
}

func (a *EndpointDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	result := DB.Model(&m.Endpoint{ID: resource.ResourceID}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("endpoint not found %v", resource)
	}

	return nil
}

func (a *EndpointDaoImpl) FindWithTenant(id *int64) (*m.Endpoint, error) {
	endpoint := &m.Endpoint{ID: *id}
	result := DB.Preload("Tenant").Find(&endpoint)

	return endpoint, result.Error
}

func (a *EndpointDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	endpoint, err := a.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(endpoint.ToEvent())

	return data, err
}
