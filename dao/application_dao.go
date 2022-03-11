package dao

import (
	"encoding/json"
	"fmt"
	"time"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// GetApplicationDao is a function definition that can be replaced in runtime in case some other DAO
// provider is needed.
var GetApplicationDao func(*int64) ApplicationDao

// getDefaultApplicationAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultApplicationDao(tenantId *int64) ApplicationDao {
	return &applicationDaoImpl{
		TenantID: tenantId,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetApplicationDao = getDefaultApplicationDao
}

type applicationDaoImpl struct {
	TenantID *int64
}

func (a *applicationDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	sourceType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	query := sourceType.HasMany(&m.Application{}, DB.Debug())

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Model(&m.Application{}).Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&applications)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return applications, count, nil
}

func (a *applicationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	query := DB.Debug().Model(&m.Application{}).
		Offset(offset).
		Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Find(&applications)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return applications, count, nil
}

func (a *applicationDaoImpl) GetById(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.First(&app)
	if result.Error != nil {
		return nil, util.NewErrNotFound("application")
	}

	return app, nil
}

func (a *applicationDaoImpl) Create(app *m.Application) error {
	app.TenantID = *a.TenantID
	result := DB.Create(app)

	return result.Error
}

func (a *applicationDaoImpl) Update(app *m.Application) error {
	result := DB.Updates(app)
	return result.Error
}

func (a *applicationDaoImpl) Delete(id *int64) (*m.Application, error) {
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

func (a *applicationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *applicationDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	application := &m.Application{ID: resource.ResourceID}
	result := DB.Preload("Source").Find(&application)

	if result.Error != nil {
		return nil, result.Error
	}

	authentication := &m.Authentication{ResourceID: application.ID,
		ResourceType:               "Application",
		ApplicationAuthentications: []m.ApplicationAuthentication{}}

	return BulkMessageFromSource(&application.Source, authentication)
}

func (a *applicationDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) error {
	result := DB.Model(&m.Application{ID: resource.ResourceID}).Updates(updateAttributes)
	if result.RowsAffected == 0 {
		return fmt.Errorf("application not found %v", resource)
	}

	return nil
}

func (a *applicationDaoImpl) FindWithTenant(id *int64) (*m.Application, error) {
	app := &m.Application{ID: *id}
	result := DB.Preload("Tenant").Find(&app)

	return app, result.Error
}

func (a *applicationDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	app, err := a.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(app.ToEvent())

	return data, err
}

func (a *applicationDaoImpl) Pause(id int64) error {
	err := DB.Debug().
		Model(&m.Application{}).
		Where("id = ?", id).
		Where("tenant_id = ?", a.TenantID).
		Update("paused_at", time.Now()).
		Error

	return err
}

func (a *applicationDaoImpl) Unpause(id int64) error {
	err := DB.Debug().
		Model(&m.Application{}).
		Where("id = ?", id).
		Where("tenant_id = ?", a.TenantID).
		Update("paused_at", nil).
		Error

	return err
}
