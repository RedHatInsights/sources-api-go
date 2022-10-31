package dao

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao/amazon"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type secretDaoSecretsManagerImpl struct {
	*RequestParams

	secretDaoDbImpl
}

func (s *secretDaoSecretsManagerImpl) Create(auth *m.Authentication) error {
	// only reach out to amazon if there is a password present, otherwise pass
	// straight through to the db dao.
	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return err
		}

		auth.TenantID = *s.TenantID
		arn, err := sm.CreateSecret(auth, *auth.Password)
		if err != nil {
			return err
		}

		err = auth.SetPassword(arn)
		if err != nil {
			return err
		}
	}

	return s.secretDaoDbImpl.Create(auth)
}

func (s *secretDaoSecretsManagerImpl) Delete(id *int64) error {
	auth, err := s.secretDaoDbImpl.GetById(id)
	if err != nil {
		return util.NewErrNotFound("secret")
	}

	if auth.Password != nil {
		sm, err := amazon.NewSecretsManagerClient()
		if err != nil {
			return err
		}

		err = sm.DeleteSecret(*auth.Password)
		if err != nil {
			return err
		}
	}

	return s.secretDaoDbImpl.Delete(id)
}

func (s *secretDaoSecretsManagerImpl) GetById(id *int64) (*m.Authentication, error) {
	auth, err := s.secretDaoDbImpl.GetById(id)
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

func (s *secretDaoSecretsManagerImpl) Update(auth *m.Authentication) error {
	if auth.Password != nil && !strings.HasPrefix(*auth.Password, config.Get().SecretsManagerPrefix) {
		// fetch the ARN of the current password (since we overwrote it in memory)
		arns := make([]*string, 1)
		err := s.getDbWithModel().
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

	return s.secretDaoDbImpl.Update(auth)
}
