package dao

import (
	"encoding/json"
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type ApplicationDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	sourceType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	query := sourceType.HasMany(&m.Application{}, DB.Debug())

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Model(&m.Application{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&applications)
	return applications, count, result.Error
}

func (a *ApplicationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	query := DB.Debug().Model(&m.Application{}).
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&applications)
	return applications, count, result.Error
}

func (a *ApplicationDaoImpl) GetById(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.First(&app)
	if result.Error != nil {
		return nil, util.NewErrNotFound("application")
	}

	return app, nil
}

func (a *ApplicationDaoImpl) Create(app *m.Application) error {
	app.TenantID = *a.TenantID
	result := DB.Create(app)

	return result.Error
}

func (a *ApplicationDaoImpl) Update(app *m.Application) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *ApplicationDaoImpl) Delete(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.Where("tenant_id = ?", a.TenantID).First(app)
	if result.Error != nil {
		return nil, util.NewErrNotFound("application")
	}

	if result := DB.Delete(app); result.Error != nil {
		return nil, fmt.Errorf("failed to delete application id %v", *id)
	}

	return app, nil
}

func (a *ApplicationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *ApplicationDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	application := &m.Application{ID: resource.ResourceID}
	result := DB.Preload("Source").Find(&application)

	if result.Error != nil {
		return nil, result.Error
	}

	return BulkMessageFromSource(&application.Source)
}

func (a *ApplicationDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	result := DB.Model(&m.Application{ID: resource.ResourceID}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("application not found %v", resource)
	}

	return nil
}

func (a *ApplicationDaoImpl) FindWithTenant(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.Preload("Tenant").Find(&app)

	return app, result.Error
}

func (a *ApplicationDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	app, err := a.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(app.ToEvent())

	return data, err
}
