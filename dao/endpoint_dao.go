package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type EndpointDaoImpl struct {
	TenantID *int64
}

func (a *EndpointDaoImpl) List(limit int, offset int, filters []middleware.Filter) ([]m.Endpoint, int64, error) {
	endpoints := make([]m.Endpoint, 0, limit)
	query := DB.Debug().
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Model(&m.Endpoint{}).Count(&count)

	result := query.Limit(limit).Find(&endpoints)
	return endpoints, count, result.Error
}

func (a *EndpointDaoImpl) GetById(id *int64) (*m.Endpoint, error) {
	app := &m.Endpoint{ID: *id}
	result := DB.First(&app)

	return app, result.Error
}

func (a *EndpointDaoImpl) Create(app *m.Endpoint) error {
	result := DB.Create(app)
	return result.Error
}

func (a *EndpointDaoImpl) Update(app *m.Endpoint) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *EndpointDaoImpl) Delete(id *int64) error {
	app := &m.Endpoint{ID: *id}
	if result := DB.Delete(app); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete endpoint id %v", *id)
	}

	return nil
}

func (a *EndpointDaoImpl) Tenant() *int64 {
	return a.TenantID
}
