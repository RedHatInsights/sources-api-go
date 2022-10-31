package dao

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao/amazon"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type authenticationSecretsManagerDaoImpl struct {
	// tenant ID + User ID + context information
	*RequestParams

	// embedded isntance of the db - since we use it for all fields except the
	// password. the only glaring difference is that we will store an
	// unencrypted ARN as the password field, so we can just fetch it directly
	// from amazon when we need to decrypt it.
	//
	// we also will continue to use all list/info methods as passthrough - only
	// overriding when necessary.
	authenticationDaoDbImpl
}

func (a *authenticationSecretsManagerDaoImpl) Create(auth *m.Authentication) error {
	// only reach out to amazon if there is a password present, otherwise pass
	// straight through to the db dao.
	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return err
		}

		auth.TenantID = *a.TenantID
		arn, err := sm.CreateSecret(auth, *auth.Password)
		if err != nil {
			return err
		}

		err = auth.SetPassword(arn)
		if err != nil {
			return err
		}
	}

	return a.authenticationDaoDbImpl.Create(auth)
}

func (a *authenticationSecretsManagerDaoImpl) BulkCreate(auth *m.Authentication) error {
	// only reach out to amazon if there is a password present, otherwise pass
	// straight through to the db dao.
	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return err
		}

		auth.TenantID = *a.TenantID
		arn, err := sm.CreateSecret(auth, *auth.Password)
		if err != nil {
			return err
		}

		err = auth.SetPassword(arn)
		if err != nil {
			return err
		}
	}

	return a.authenticationDaoDbImpl.BulkCreate(auth)
}

func (a *authenticationSecretsManagerDaoImpl) Update(auth *m.Authentication) error {
	// only reach out to amazon if there is a password present, otherwise pass
	// straight through to the db dao.
	if auth.Password != nil && !strings.HasPrefix(*auth.Password, config.Get().SecretsManagerPrefix) {
		// fetch the ARN of the current password (since we overwrote it in memory)
		arns := make([]*string, 1)
		err := a.getDbWithModel().
			Where("id = ?", auth.GetID()).
			Limit(1).
			Pluck("password_hash", &arns).Error
		if err != nil {
			return err
		}
		if *arns[0] == "" {
			return fmt.Errorf("failed to fetch ARN for authentication %v", auth.GetID())
		}

		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return err
		}

		err = sm.UpdateSecret(*arns[0], *auth.Password)
		if err != nil {
			return err
		}

		// set the password back to the ARN so that we don't overwrite it.
		auth.Password = arns[0]
	}

	return a.authenticationDaoDbImpl.Update(auth)
}

func (a *authenticationSecretsManagerDaoImpl) Delete(id string) (*m.Authentication, error) {
	auth, err := a.authenticationDaoDbImpl.Delete(id)
	if err != nil {
		return nil, err
	}

	// only reach out to amazon to nuke the secret if the password exists
	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return nil, err
		}

		err = sm.DeleteSecret(*auth.Password)
		if err != nil {
			return nil, err
		}
	}

	return auth, nil
}

// BulkDelete deletes all the authentications given as a list, and returns the ones that were deleted.
func (a *authenticationSecretsManagerDaoImpl) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	auths, err := a.authenticationDaoDbImpl.BulkDelete(authentications)
	if err != nil {
		return nil, err
	}

	sm, err := amazon.NewSecretsManagerClient()
	if err != nil {
		return nil, err
	}

	// go through and clean up the aws resoures, logging if there is a failure.
	for i := range authentications {
		if authentications[i].Password != nil {
			err := sm.DeleteSecret(*authentications[i].Password)
			if err != nil {
				DB.Logger.Warn(a.ctx, "Failed to delete secret %v: %v", authentications[i].Password, err)
			}
		}
	}

	return auths, nil
}

// overriding the GetById function to fetch from secrets manager if the password
// is present - this is mostly for internal responses.
func (a *authenticationSecretsManagerDaoImpl) GetById(id string) (*m.Authentication, error) {
	auth, err := a.authenticationDaoDbImpl.GetById(id)
	if err != nil {
		return nil, err
	}

	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return nil, err
		}

		pass, err := sm.GetSecret(*auth.Password)
		if err != nil {
			return nil, err
		}

		// set the password in-memory to the secrets-manager password
		auth.Password = pass
	}

	return auth, nil
}
