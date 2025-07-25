package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetApplicationAuthenticationDao is a function definition that can be replaced in runtime in case some other DAO
// provider is needed.
var GetApplicationAuthenticationDao func(*RequestParams) ApplicationAuthenticationDao

// getDefaultApplicationAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultApplicationAuthenticationDao(daoParams *RequestParams) ApplicationAuthenticationDao {
	return &applicationAuthenticationDaoImpl{RequestParams: daoParams}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetApplicationAuthenticationDao = getDefaultApplicationAuthenticationDao
}

type applicationAuthenticationDaoImpl struct {
	*RequestParams
}

func (a applicationAuthenticationDaoImpl) getDb() *gorm.DB {
	if a.TenantID == nil {
		panic("nil tenant found in applicationAuthentication db DAO")
	}

	query := DB.Debug().WithContext(a.ctx)
	query = query.Where("tenant_id = ?", a.TenantID)

	if a.UserID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", a.UserID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	return query
}

func (a applicationAuthenticationDaoImpl) getDbWithModel() *gorm.DB {
	return a.getDb().Model(&m.ApplicationAuthentication{})
}

func (a *applicationAuthenticationDaoImpl) ApplicationAuthenticationsByApplications(applications []m.Application) ([]m.ApplicationAuthentication, error) {
	var applicationAuthentications []m.ApplicationAuthentication

	applicationIDs := make([]int64, 0)
	for _, value := range applications {
		applicationIDs = append(applicationIDs, value.ID)
	}

	err := a.getDb().
		Preload("Tenant").
		Where("application_id IN ?", applicationIDs).
		Where("tenant_id = ?", a.TenantID).
		Find(&applicationAuthentications).
		Error
	if err != nil {
		return nil, err
	}

	return applicationAuthentications, nil
}

func (a *applicationAuthenticationDaoImpl) ApplicationAuthenticationsByAuthentications(authentications []m.Authentication) ([]m.ApplicationAuthentication, error) {
	var applicationAuthentications []m.ApplicationAuthentication

	query := a.getDb().Preload("Tenant")

	switch config.Get().SecretStore {
	case config.DatabaseStore, config.SecretsManagerStore:
		authIds := make([]int64, len(authentications))

		for _, value := range authentications {
			authIds = append(authIds, value.DbID)
		}

		query.Where("authentication_id IN ?", authIds)

	case config.VaultStore:
		authUuids := make([]string, len(authentications))

		for _, value := range authentications {
			authUuids = append(authUuids, value.ID)
		}

		query.Where("authentication_uid IN ?", authUuids)

	default:
		return nil, m.ErrBadSecretStore
	}

	err := query.
		Where("tenant_id = ?", a.TenantID).
		Find(&applicationAuthentications).
		Error
	if err != nil {
		return nil, err
	}

	return applicationAuthentications, nil
}

func (a *applicationAuthenticationDaoImpl) ApplicationAuthenticationsByResource(resourceType string, applications []m.Application, authentications []m.Authentication) ([]m.ApplicationAuthentication, error) {
	if resourceType == "Source" {
		return a.ApplicationAuthenticationsByApplications(applications)
	}

	return a.ApplicationAuthenticationsByAuthentications(authentications)
}

func (a *applicationAuthenticationDaoImpl) List(limit int, offset int, filters []util.Filter) ([]m.ApplicationAuthentication, int64, error) {
	appAuths := make([]m.ApplicationAuthentication, 0, limit)
	query := a.getDbWithModel().Where("tenant_id = ?", a.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	result := query.Limit(limit).Offset(offset).Find(&appAuths)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return appAuths, count, nil
}

func (a *applicationAuthenticationDaoImpl) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	var appAuth m.ApplicationAuthentication

	err := a.getDbWithModel().
		Where("id = ?", *id).
		First(&appAuth).
		Error
	if err != nil {
		return nil, util.NewErrNotFound("application authentication")
	}

	return &appAuth, nil
}

func (a *applicationAuthenticationDaoImpl) Create(appAuth *m.ApplicationAuthentication) error {
	appAuth.TenantID = *a.TenantID

	err := DB.Debug().Create(appAuth).Error
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID}).Errorf(`Unable to create application authentication: %s`, err)

		return util.NewErrBadRequest("failed to create application_authentication: " + err.Error())
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID, "application_authentication_id": appAuth.ID}).Info("Application authentication created")
		return nil
	}
}

func (a *applicationAuthenticationDaoImpl) Update(appAuth *m.ApplicationAuthentication) error {
	err := a.getDb().Updates(appAuth).Error
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID, "application_authentication_id": appAuth.ID}).Errorf(`Unable to update application authentication: %s`, err)

		return err
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID, "application_authentication_id": appAuth.ID}).Info("Application authentication updated")
		return nil
	}
}

func (a *applicationAuthenticationDaoImpl) Delete(id *int64) (*m.ApplicationAuthentication, error) {
	var applicationAuthentication m.ApplicationAuthentication

	result := a.getDb().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Where("tenant_id = ?", a.TenantID).
		Delete(&applicationAuthentication)

	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID, "application_authentication_id": id}).Errorf(`Unable to delete application authentication: %s`, result.Error)

		return nil, fmt.Errorf(`failed to delete application authentication with id "%d": %s`, id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, util.NewErrNotFound("application authentication")
	}

	logger.Log.WithFields(logrus.Fields{"tenant_id": *a.TenantID, "application_authentication_id": id}).Info("Application authentication deleted")

	return &applicationAuthentication, nil
}

func (a *applicationAuthenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}
