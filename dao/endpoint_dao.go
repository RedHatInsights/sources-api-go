package dao

import (
	"encoding/json"
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm/clause"
)

// GetEndpointDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetEndpointDao func(*int64) EndpointDao

// getDefaultAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultEndpointDao(tenantId *int64) EndpointDao {
	return &endpointDaoImpl{
		TenantID: tenantId,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetEndpointDao = getDefaultEndpointDao
}

type endpointDaoImpl struct {
	TenantID *int64
}

func (a *endpointDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	relationObject, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
	}

	query := relationObject.HasMany(&m.Endpoint{}, DB.Debug())
	query = query.Where("endpoints.tenant_id = ?", a.TenantID)

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Model(&m.Endpoint{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&endpoints)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return endpoints, count, nil
}

func (a *endpointDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	query := DB.Debug().Model(&m.Endpoint{}).
		Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&endpoints)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return endpoints, count, nil
}

// GetByIdWithPreload searches for an application and preloads any specified relations.
func (a *endpointDaoImpl) GetByIdWithPreload(id *int64, preloads ...string) (*m.Endpoint, error) {
	q := DB.Debug().
		Model(&m.Endpoint{}).
		Where("id = ?", *id).
		Where("tenant_id = ?", a.TenantID)

	for _, preload := range preloads {
		q = q.Preload(preload)
	}

	var endpoint m.Endpoint
	err := q.
		First(&endpoint).
		Error

	return &endpoint, err
}

func (a *endpointDaoImpl) GetById(id *int64) (*m.Endpoint, error) {
	var endpoint m.Endpoint

	err := DB.Debug().
		Model(&m.Endpoint{}).
		Where("id = ?", *id).
		Where("tenant_id = ?", a.TenantID).
		First(&endpoint).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("endpoint")
	}

	return &endpoint, nil
}

func (a *endpointDaoImpl) Create(app *m.Endpoint) error {
	app.TenantID = *a.TenantID

	result := DB.Debug().Create(app)
	return result.Error
}

func (a *endpointDaoImpl) Update(app *m.Endpoint) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *endpointDaoImpl) Delete(id *int64) (*m.Endpoint, error) {
	var endpoint m.Endpoint

	result := DB.
		Debug().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Where("tenant_id = ?", a.TenantID).
		Delete(&endpoint)

	if result.Error != nil {
		return nil, fmt.Errorf(`failed to delete endpoint with id "%d": %s`, id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, util.NewErrNotFound("endpoint")
	}

	return &endpoint, nil
}

func (a *endpointDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *endpointDaoImpl) CanEndpointBeSetAsDefaultForSource(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	// add double quotes to the "default" column to avoid any clashes with postgres' "default" keyword
	result := DB.Debug().Where(`"default" = true AND source_id = ?`, sourceId).First(&endpoint)
	return result.Error != nil
}

func (a *endpointDaoImpl) IsRoleUniqueForSource(role string, sourceId int64) bool {
	endpoint := &m.Endpoint{}
	result := DB.Debug().Where("role = ? AND source_id = ?", role, sourceId).First(&endpoint)

	// If the record doesn't exist "result.Error" will have a "record not found" error
	return result.Error != nil
}

func (a *endpointDaoImpl) SourceHasEndpoints(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	result := DB.Debug().Where("source_id = ?", sourceId).First(&endpoint)

	return result.Error == nil
}

func (a *endpointDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	var endpoint m.Endpoint

	err := DB.Debug().
		Model(&m.Endpoint{}).
		Where("id = ?", resource.ResourceID).
		Preload("Source").
		Find(&endpoint).
		Error

	if err != nil {
		return nil, err
	}

	authentication := &m.Authentication{ResourceID: endpoint.ID, ResourceType: "Endpoint", ApplicationAuthentications: []m.ApplicationAuthentication{}}

	return BulkMessageFromSource(&endpoint.Source, authentication)
}

func (a *endpointDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	result := DB.
		Debug().
		Model(&m.Endpoint{}).
		Where("id = ?", resource.ResourceID).
		Updates(updateAttributes)

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("endpoint not found %v", resource)
	}

	a.TenantID = &resource.TenantID
	endpoint, err := a.GetByIdWithPreload(&resource.ResourceID, "Source")
	if err != nil {
		return nil, err
	}

	return endpoint, nil
}

func (a *endpointDaoImpl) FindWithTenant(id *int64) (*m.Endpoint, error) {
	var endpoint m.Endpoint

	err := DB.Debug().
		Model(&m.Endpoint{}).
		Where("id = ?", *id).
		Preload("Tenant").
		Find(&endpoint).
		Error

	return &endpoint, err
}

func (a *endpointDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	endpoint, err := a.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(endpoint.ToEvent())

	return data, err
}

func (a *endpointDaoImpl) Exists(endpointId int64) (bool, error) {
	var endpointExists bool

	err := DB.Model(&m.Endpoint{}).
		Select("1").
		Where("id = ?", endpointId).
		Where("tenant_id = ?", a.TenantID).
		Scan(&endpointExists).
		Error

	if err != nil {
		return false, err
	}

	return endpointExists, nil
}
