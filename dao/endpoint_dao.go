package dao

import (
	"encoding/json"
	"fmt"
	"strconv"

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

func (e *endpointDaoImpl) Exists(id *int64) (bool, error) {
	var exists bool

	result := DB.Debug().
		Select("1").
		Model(&m.Endpoint{ID: *id}).
		Where("tenant_id = ?", e.TenantID).
		First(&exists)

	return exists, result.Error
}

func (e *endpointDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	query := DB.Debug().Model(&m.Endpoint{}).
		Offset(offset).
		Where("tenant_id = ?", e.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&endpoints)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return endpoints, count, nil
}

func (e *endpointDaoImpl) ListForSource(sourceID *int64, limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	exists, err := GetSourceDao(e.TenantID).Exists(sourceID)
	if !exists || err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	return e.List(limit, offset, append(filters, util.Filter{Name: "source_id", Value: []string{strconv.FormatInt(*sourceID, 10)}}))
}

func (e *endpointDaoImpl) GetById(id *int64) (*m.Endpoint, error) {
	app := &m.Endpoint{ID: *id}
	result := DB.Debug().First(&app)
	if result.Error != nil {
		return nil, util.NewErrNotFound("endpoint")
	}

	return app, nil
}

func (e *endpointDaoImpl) Create(app *m.Endpoint) error {
	app.TenantID = *e.TenantID

	result := DB.Debug().Create(app)
	return result.Error
}

func (e *endpointDaoImpl) Update(app *m.Endpoint) error {
	result := DB.Updates(app)
	return result.Error
}

func (e *endpointDaoImpl) Delete(id *int64) (*m.Endpoint, error) {
	var endpoint m.Endpoint

	result := DB.
		Debug().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Where("tenant_id = ?", e.TenantID).
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

func (e *endpointDaoImpl) CanEndpointBeSetAsDefaultForSource(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	// add double quotes to the "default" column to avoid any clashes with postgres' "default" keyword
	result := DB.Debug().Where(`"default" = true AND source_id = ?`, sourceId).First(&endpoint)
	return result.Error != nil
}

func (e *endpointDaoImpl) IsRoleUniqueForSource(role string, sourceId int64) bool {
	endpoint := &m.Endpoint{}
	result := DB.Debug().Where("role = ? AND source_id = ?", role, sourceId).First(&endpoint)

	// If the record doesn't exist "result.Error" will have a "record not found" error
	return result.Error != nil
}

func (e *endpointDaoImpl) SourceHasEndpoints(sourceId int64) bool {
	endpoint := &m.Endpoint{}

	result := DB.Debug().Where("source_id = ?", sourceId).First(&endpoint)

	return result.Error == nil
}

func (e *endpointDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	endpoint := &m.Endpoint{ID: resource.ResourceID}
	result := DB.Debug().Preload("Source").Find(&endpoint)

	if result.Error != nil {
		return nil, result.Error
	}

	authentication := &m.Authentication{ResourceID: endpoint.ID, ResourceType: "Endpoint", ApplicationAuthentications: []m.ApplicationAuthentication{}}
	return BulkMessageFromSource(&endpoint.Source, authentication)
}

func (e *endpointDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	result := DB.Debug().Model(&m.Endpoint{ID: resource.ResourceID}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("endpoint not found %v", resource)
	}

	return nil
}

func (e *endpointDaoImpl) FindWithTenant(id *int64) (*m.Endpoint, error) {
	endpoint := &m.Endpoint{ID: *id}
	result := DB.Debug().Preload("Tenant").Find(&endpoint)

	return endpoint, result.Error
}

func (e *endpointDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	endpoint, err := e.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(endpoint.ToEvent())

	return data, err
}
