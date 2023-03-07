package dao

import (
	"context"
	"fmt"

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

func (u *userDaoImpl) FindOrCreate(ctx context.Context, userID string) (*m.User, error) {
	var user m.User

	if u.TenantID == nil {
		return nil, fmt.Errorf("tenant id is missing to call FindOrCreate")
	}

	err := DB.WithContext(ctx).Model(&m.User{}).
		Where("user_id = ?", userID).
		Where("tenant_id = ?", *u.TenantID).
		First(&user).
		Error

	if err != nil {
		user.TenantID = *u.TenantID
		user.UserID = userID
		err = DB.WithContext(ctx).Debug().Create(&user).Error
		if err != nil {
			return nil, err
		}
	}

	return &user, nil
}
