package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type ApplicationAuthenticationDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationAuthenticationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.ApplicationAuthentication, int64, error) {
	applications := make([]m.ApplicationAuthentication, 0, limit)
	query := DB.Debug().Model(&m.ApplicationAuthentication{}).
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&appAuths)
	return appAuths, count, result.Error
}

func (a *ApplicationAuthenticationDaoImpl) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	appAuth := &m.ApplicationAuthentication{ID: *id}
	result := DB.First(&appAuth)

	return appAuth, result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Create(appAuth *m.ApplicationAuthentication) error {
	result := DB.Create(appAuth)
	return result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Update(appAuth *m.ApplicationAuthentication) error {
	result := DB.Updates(appAuth)
	return result.Error
}

func (a *ApplicationAuthenticationDaoImpl) Delete(id *int64) error {
	appAuth := &m.ApplicationAuthentication{ID: *id}
	if result := DB.Delete(appAuth); result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete application id %v", *id)
	}

	return nil
}

func (a *ApplicationAuthenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}
