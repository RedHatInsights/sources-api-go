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

func (u *userDaoImpl) findOrCreate(user *m.User) (*m.User, error) {
	var foundUser m.User

	err := DB.Model(&m.User{}).
		Where("user_id = ?", user.UserID).
		Where("tenant_id = ?", *u.TenantID).
		Find(&foundUser).
		Error

	if err != nil {
		return nil, err
	}

	if foundUser.Id == 0 {
		user.TenantID = *u.TenantID
		resultError := DB.Debug().Create(user).Error
		if resultError != nil {
			return nil, resultError
		}

		return user, nil
	} else {
		return &foundUser, nil
	}
}

func (u *userDaoImpl) FindOrCreateUserIfResourceOwnershipActive(userResource *m.UserResource) error {
	if userResource.UserOwnershipActive() {
		user, err := u.findOrCreate(userResource.User)
		if err != nil {
			return err
		}

		userResource.User = user
	}

	return nil
}
