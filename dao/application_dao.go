package dao

import (
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type ApplicationDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []middleware.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	sourceType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
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

func (a *ApplicationDaoImpl) List(limit int, offset int, filters []middleware.Filter) ([]m.Application, int64, error) {
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

	return app, result.Error
}

func (a *ApplicationDaoImpl) Create(app *m.Application) error {
	result := DB.Create(app)
	return result.Error
}

func (a *ApplicationDaoImpl) Update(app *m.Application) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *ApplicationDaoImpl) Delete(id *int64) error {
	app := &m.Application{ID: *id}
	if result := DB.Delete(app); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete application id %v", *id)
	}

	return nil
}

func (a *ApplicationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *ApplicationDaoImpl) BulkMessage(id *int64) (map[string]interface{}, error) {
	application := &m.Application{ID: *id}
	resource := DB.Preload("Source.Tenant").Preload("Source.Applications.Tenant").Preload("Source.Endpoints.Tenant").Find(&application)

	if resource.Error != nil {
		return nil, resource.Error
	}

	bulkMessage := map[string]interface{}{}
	bulkMessage["source"] = application.Source.ToEvent()

	endpoints := make([]m.EndpointEvent, len(application.Source.Endpoints))
	for i, endpoint := range application.Source.Endpoints {
		endpoints[i] = *endpoint.ToEvent()
	}

	bulkMessage["endpoints"] = endpoints

	applications := make([]m.ApplicationEvent, len(application.Source.Applications))
	for i, application := range application.Source.Applications {
		applications[i] = *application.ToEvent()
	}

	bulkMessage["applications"] = applications

	bulkMessage["authentications"] = []m.Authentication{}
	bulkMessage["application_authentications"] = []m.ApplicationAuthenticationEvent{}

	return bulkMessage, nil
}

func (a *ApplicationDaoImpl) FetchAndUpdateBy(id *int64, updateAttributes map[string]interface{}) error {
	src, err := a.GetById(id)
	if err != nil {
		err = DB.Model(src).Updates(updateAttributes).Error
	}

	return err
}

func (a *ApplicationDaoImpl) FindWithTenant(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.Preload("Tenant").Find(&app)

	return app, result.Error
}

func (a *ApplicationDaoImpl) ToEventJSON(id *int64) ([]byte, error) {
	app, err := a.FindWithTenant(id)
	data, _ := json.Marshal(app.ToEvent())

	return data, err
}
