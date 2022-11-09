package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

type secretDaoDbImpl struct {
	*RequestParams
}

func (secret *secretDaoDbImpl) getDb() *gorm.DB {
	if secret.TenantID == nil {
		panic("nil tenant found in sourceDaoImpl DAO")
	}

	query := DB.Debug().WithContext(secret.ctx)
	query = query.Where("tenant_id = ?", secret.TenantID)
	query = query.Where("resource_type = ?", secretResourceType)

	if secret.UserID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", secret.UserID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	return query
}

func (secret *secretDaoDbImpl) getDbWithModel() *gorm.DB {
	return secret.getDb().Model(&m.Authentication{})
}

func (secret *secretDaoDbImpl) GetById(id *int64) (*m.Authentication, error) {
	var secretAuthentication m.Authentication

	err := secret.getDbWithModel().
		Where("id = ?", id).
		First(&secretAuthentication).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("secret")
	}

	return &secretAuthentication, nil
}

func (secret *secretDaoDbImpl) Create(authentication *m.Authentication) error {
	authentication.TenantID = *secret.TenantID // the TenantID gets injected in the middleware
	authentication.ResourceType = secretResourceType
	authentication.ResourceID = *secret.TenantID

	return DB.
		Debug().
		Create(authentication).
		Error
}

func (secret *secretDaoDbImpl) Delete(id *int64) error {
	var authentication m.Authentication

	err := secret.getDb().
		Where("id = ?", id).
		First(&authentication).
		Error

	if err != nil {
		return util.NewErrNotFound("secret")
	}

	err = secret.getDb().
		Delete(&authentication).
		Error

	if err != nil {
		return fmt.Errorf(`failed to delete secret with id "%d"`, &id)
	}

	return nil
}

func (secret *secretDaoDbImpl) NameExistsInCurrentTenant(name string) bool {
	err := secret.getDbWithModel().
		Where("name = ?", name).
		First(&m.Authentication{}).
		Error

	return err == nil
}

func (secret *secretDaoDbImpl) List(limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	query := secret.getDbWithModel()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	count := int64(0)
	query.Count(&count)

	secrets := make([]m.Authentication, 0, limit)
	err = query.
		Limit(limit).
		Offset(offset).
		Find(&secrets).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return secrets, count, nil
}

func (secret *secretDaoDbImpl) Update(authentication *m.Authentication) error {
	return secret.getDb().
		Updates(authentication).
		Error
}
