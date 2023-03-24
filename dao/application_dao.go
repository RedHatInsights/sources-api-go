package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/jackc/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetApplicationDao is a function definition that can be replaced in runtime in case some other DAO
// provider is needed.
var GetApplicationDao func(*RequestParams) ApplicationDao

// getDefaultApplicationAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultApplicationDao(daoParams *RequestParams) ApplicationDao {
	return &applicationDaoImpl{RequestParams: daoParams}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetApplicationDao = getDefaultApplicationDao
}

type applicationDaoImpl struct {
	*RequestParams
}

func (a *applicationDaoImpl) getDbWithTable(query *gorm.DB, table string) *gorm.DB {
	if a.TenantID == nil {
		panic("nil tenant found in sourceDaoImpl DAO")
	}

	var whereCondition string
	if table != "" {
		whereCondition = fmt.Sprintf("%s.", table)
	}

	query = query.Where(whereCondition+"tenant_id = ?", a.TenantID)

	return a.useUserForDB(query, table)
}

func (a *applicationDaoImpl) useUserForDB(query *gorm.DB, table string) *gorm.DB {
	var whereCondition string
	if table != "" {
		whereCondition = fmt.Sprintf("%s.", table)
	}

	if a.UserID != nil {
		condition := fmt.Sprintf("%[1]vuser_id IS NULL OR %[1]vuser_id = ?", whereCondition)
		query = query.Where(condition, a.UserID)
	} else {
		query = query.Where(whereCondition + "user_id IS NULL")
	}

	return query
}

func (a *applicationDaoImpl) getDb() *gorm.DB {
	return a.getDbWithTable(DB.Debug().WithContext(a.ctx), "")
}

func (a *applicationDaoImpl) getDbWithModel() *gorm.DB {
	return a.getDb().Model(&m.Application{})
}

func (a *applicationDaoImpl) SubCollectionList(primaryCollection interface{}, limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	applications := make([]m.Application, 0, limit)
	relationObject, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
	}

	query := relationObject.HasMany(&m.Application{}, DB.Debug())
	query = a.getDbWithTable(query, "applications")

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
	query := a.getDbWithModel()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.
		Limit(limit).
		Offset(offset).
		Find(&applications)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}
	return applications, count, nil
}

func (a *applicationDaoImpl) GetById(id *int64) (*m.Application, error) {
	var app m.Application

	err := a.getDbWithModel().
		Where("id = ?", *id).
		First(&app).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("application")
	}

	return &app, nil
}

// GetByIdWithPreload searches for an application and preloads any specified relations.
func (a *applicationDaoImpl) GetByIdWithPreload(id *int64, preloads ...string) (*m.Application, error) {
	q := a.getDbWithModel().
		Where("id = ?", *id)

	for _, preload := range preloads {
		q = q.Preload(preload)
	}

	var app m.Application
	err := q.
		First(&app).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("application")
	}

	return &app, nil
}

func (a *applicationDaoImpl) Create(app *m.Application) error {
	app.TenantID = *a.TenantID
	result := DB.Debug().Create(app)

	// Check if specific error code is returned
	var pgErr *pgconn.PgError
	if errors.As(result.Error, &pgErr) {
		// unique constraint violation for index (source id + app type id + tenant id)
		if pgErr.Code == PgUniqueConstraintViolation && strings.Contains(pgErr.Detail, "Key (source_id, application_type_id, tenant_id)") {
			message := fmt.Sprintf("Application of application type = %d already exists for the source id = %d", app.ApplicationTypeID, app.SourceID)
			return util.NewErrBadRequest(message)
		}
	}
	return result.Error
}

func (a *applicationDaoImpl) Update(app *m.Application) error {
	result := a.getDb().Omit(clause.Associations).Updates(app)
	return result.Error
}

func (a *applicationDaoImpl) Delete(id *int64) (*m.Application, error) {
	var application m.Application

	result := a.getDb().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Delete(&application)

	if result.Error != nil {
		return nil, fmt.Errorf(`failed to delete application with id "%d": %s`, id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, util.NewErrNotFound("application")
	}

	return &application, nil
}

func (a *applicationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *applicationDaoImpl) User() *int64 {
	return a.UserID
}

func (a *applicationDaoImpl) IsSuperkey(id int64) bool {
	var valid bool

	result := a.getDbWithTable(DB.Debug(), "applications").
		Model(&m.Application{}).
		Select(`"Source".app_creation_workflow = ?`, m.AccountAuth).
		Where("applications.id = ?", id).
		Joins("Source").
		First(&valid)

	if result.Error != nil {
		DB.Logger.Warn(a.ctx, "Failed to determine if app %v is superkey: %v", id, result.Error)
		return false
	}

	return valid
}

func (a *applicationDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	var application m.Application

	err := a.getDbWithModel().
		Where("id = ?", resource.ResourceID).
		Preload("Source").
		Find(&application).
		Error

	if err != nil {
		return nil, err
	}

	authentication := &m.Authentication{ResourceID: application.ID,
		ResourceType:               "Application",
		ApplicationAuthentications: []m.ApplicationAuthentication{}}

	return BulkMessageFromSource(&application.Source, authentication)
}

func (a *applicationDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	result := a.getDbWithModel().
		Where("id = ?", resource.ResourceID).
		Updates(updateAttributes)

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("application not found %v", resource)
	}

	application, err := a.GetByIdWithPreload(&resource.ResourceID, "Source")
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (a *applicationDaoImpl) FindWithTenant(id *int64) (*m.Application, error) {
	var app m.Application

	err := a.getDbWithModel().
		Where("id = ?", *id).
		Preload("Tenant").
		Find(&app).
		Error

	return &app, err
}

func (a *applicationDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	app, err := a.FindWithTenant(&resource.ResourceID)
	data, _ := json.Marshal(app.ToEvent())

	return data, err
}

func (a *applicationDaoImpl) Pause(id int64) error {
	err := a.getDbWithModel().
		Where("id = ?", id).
		Update("paused_at", time.Now()).
		Error

	return err
}

func (a *applicationDaoImpl) Unpause(id int64) error {
	err := a.getDbWithModel().
		Where("id = ?", id).
		Update("paused_at", nil).
		Error

	return err
}

func (a *applicationDaoImpl) DeleteCascade(applicationId int64) ([]m.ApplicationAuthentication, *m.Application, error) {
	var applicationAuthentications []m.ApplicationAuthentication
	var application *m.Application

	// The application authentications are fetched with the "Tenant" table preloaded, so that all the objects are
	// returned with the "external_tenant" column populated. This is required to be able to raise events with the
	// "tenant" key populated.
	//
	// The "len(objects) != 0" check to delete the resources is necessary to avoid Gorm issuing the "cannot batch
	// delete without a where condition" error, since there might be times when the applications don't have any related
	// application authentications.
	err := DB.
		Debug().
		Transaction(func(tx *gorm.DB) error {
			// Fetch and delete the application authentications.
			err := tx.
				Model(m.ApplicationAuthentication{}).
				Preload("Tenant").
				Where("application_id = ?", applicationId).
				Where("tenant_id = ?", a.TenantID).
				Find(&applicationAuthentications).
				Error

			if err != nil {
				return err
			}

			if len(applicationAuthentications) != 0 {
				err = tx.
					Delete(&applicationAuthentications).
					Error

				if err != nil {
					return err
				}
			}

			// Fetch and delete the application itself.
			err = tx.
				Model(m.Application{}).
				Preload("Tenant").
				Where("id = ?", applicationId).
				Where("tenant_id = ?", a.TenantID).
				Find(&application).
				Error

			if application != nil {
				err = tx.
					Delete(&application).
					Error
			}

			return err
		})

	if err != nil {
		return nil, nil, err
	}

	return applicationAuthentications, application, err
}

func (a *applicationDaoImpl) Exists(applicationId int64) (bool, error) {
	var applicationExists bool

	err := a.getDbWithModel().
		Select("1").
		Where("id = ?", applicationId).
		Scan(&applicationExists).
		Error

	if err != nil {
		return false, err
	}

	return applicationExists, nil
}
