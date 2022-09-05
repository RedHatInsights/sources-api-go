package mocks

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockApplicationAuthenticationDao struct {
	ApplicationAuthentications []m.ApplicationAuthentication
}

func (mockAppAuthDao MockApplicationAuthenticationDao) List(_, _ int, _ []util.Filter) ([]m.ApplicationAuthentication, int64, error) {
	count := int64(len(mockAppAuthDao.ApplicationAuthentications))
	return mockAppAuthDao.ApplicationAuthentications, count, nil
}

func (mockAppAuthDao MockApplicationAuthenticationDao) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	for _, appAuth := range mockAppAuthDao.ApplicationAuthentications {
		if appAuth.ID == *id {
			return &appAuth, nil
		}
	}

	return nil, util.NewErrNotFound("application authentication")
}

func (mockAppAuthDao MockApplicationAuthenticationDao) Create(_ *m.ApplicationAuthentication) error {
	return nil
}

func (mockAppAuthDao MockApplicationAuthenticationDao) Update(_ *m.ApplicationAuthentication) error {
	panic("implement me")
}

func (mockAppAuthDao MockApplicationAuthenticationDao) Delete(id *int64) (*m.ApplicationAuthentication, error) {
	return mockAppAuthDao.GetById(id)
}

func (mockAppAuthDao MockApplicationAuthenticationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (mockAppAuthDao MockApplicationAuthenticationDao) ApplicationAuthenticationsByResource(_ string, _ []m.Application, _ []m.Authentication) ([]m.ApplicationAuthentication, error) {
	return mockAppAuthDao.ApplicationAuthentications, nil
}
