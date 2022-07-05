package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

var GetUserDao func(*int64) UserDao

// getDefaultUserDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultUserDao(tenantId *int64) UserDao {
	return &userDaoImpl{
		TenantID: tenantId,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetUserDao = getDefaultUserDao
}

type userDaoImpl struct {
	TenantID *int64
}

func (u *userDaoImpl) create(user *m.User) error {
	var userExists bool

	err := DB.Model(&m.User{}).
		Select("1").
		Where("user_id = ?", user.UserID).
		Where("tenant_id = ?", *u.TenantID).
		Scan(&userExists).
		Error

	if err != nil {
		return err
	}

	if !userExists {
		user.TenantID = *u.TenantID
		result := DB.Debug().Create(user)
		return result.Error
	} else {
		return nil
	}
}

func (u *userDaoImpl) CreateIfResourceOwnershipActive(userResource *m.UserResource) error {
	if userResource.UserOwnershipActive() {
		err := u.create(userResource.User)
		if err != nil {
			return err
		}
	}

	return nil
}
