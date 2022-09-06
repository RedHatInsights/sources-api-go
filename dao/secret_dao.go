package dao

import (
	"context"
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

const secretResourceType = "Tenant"

var GetSecretDao func(daoParams *RequestParams) SecretDao

type secretDaoDbImpl struct {
	TenantID *int64
	UserID   *int64
	ctx      context.Context
}

func getDefaultSecretDao(daoParams *RequestParams) SecretDao {
	var tenantID, userID *int64
	var ctx context.Context
	if daoParams != nil && daoParams.TenantID != nil {
		tenantID = daoParams.TenantID
		userID = daoParams.UserID
		ctx = daoParams.ctx
	}

	return &secretDaoDbImpl{
		TenantID: tenantID,
		UserID:   userID,
		ctx:      ctx,
	}
}

func init() {
	GetSecretDao = getDefaultSecretDao
}

func (secret *secretDaoDbImpl) getDb() *gorm.DB {
	if secret.TenantID == nil {
		panic("nil tenant found in sourceDaoImpl DAO")
	}

	query := DB.Debug().WithContext(secret.ctx)
	query = query.Where("tenant_id = ?", secret.TenantID)

	query = query.Where("resource_type = ?", secretResourceType)

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
	if authentication.Password != nil {
		encryptedValue, err := util.Encrypt(*authentication.Password)
		if err != nil {
			return err
		}

		authentication.Password = &encryptedValue
	}

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
