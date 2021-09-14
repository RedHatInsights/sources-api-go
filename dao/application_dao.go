package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type ApplicationDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationDaoImpl) List(limit int, offset int, filters []middleware.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	query := DB.Debug().
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Model(&m.Application{}).Count(&count)

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
