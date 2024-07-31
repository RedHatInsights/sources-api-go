package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	m "github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/gorm/clause"
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

func (u *userDaoImpl) FindOrCreate(userID string) (*m.User, error) {
	var user m.User

	if u.TenantID == nil {
		return nil, fmt.Errorf("tenant id is missing to call FindOrCreate")
	}

	err := DB.Model(&m.User{}).
		Where("user_id = ?", userID).
		Where("tenant_id = ?", *u.TenantID).
		First(&user).
		Error

	if err != nil {
		user.TenantID = *u.TenantID
		user.UserID = userID

		stmt := DB.Debug()

		if config.Get().HandleTenantRefresh {
			// tenant refresh = we run into conflicts where one user_id is tied to another tenant (since the org_id changes), therefore we tack an `ON CONFLICT UPDATE` if we're in an environment where this can happen
			stmt = stmt.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"tenant_id"}),
			})
		}

		err = stmt.Create(&user).Error
		if err != nil {
			return nil, err
		}
	}

	return &user, nil
}
