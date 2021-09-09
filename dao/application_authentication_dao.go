package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type ApplicationAuthenticationDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationAuthenticationDaoImpl) List(limit int, offset int, filters []middleware.Filter) ([]m.ApplicationAuthentication, int64, error) {
	applications := make([]m.ApplicationAuthentication, 0, limit)
	query := DB.Debug().Model(&m.ApplicationAuthentication{}).
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&applications)
	return applications, count, result.Error
}

func (a *ApplicationAuthenticationDaoImpl) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	app := &m.ApplicationAuthentication{ID: *id}
	result := DB.First(&app)

	return app, result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Create(app *m.ApplicationAuthentication) error {
	result := DB.Create(app)
	return result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Update(app *m.ApplicationAuthentication) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Delete(id *int64) error {
	app := &m.ApplicationAuthentication{ID: *id}
	if result := DB.Delete(app); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete application id %v", *id)
	}

	return nil
}

func (a *ApplicationAuthenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}
