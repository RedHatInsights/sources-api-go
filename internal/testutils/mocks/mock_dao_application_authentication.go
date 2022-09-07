package mocks

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockApplicationAuthenticationDao struct {
	ApplicationAuthentications []m.ApplicationAuthentication
}

func (m MockApplicationAuthenticationDao) List(limit, offset int, filters []util.Filter) ([]m.ApplicationAuthentication, int64, error) {
	count := int64(len(m.ApplicationAuthentications))
	return m.ApplicationAuthentications, count, nil
}

func (m MockApplicationAuthenticationDao) GetById(id *int64) (*m.ApplicationAuthentication, error) {
	for _, appAuth := range m.ApplicationAuthentications {
		if appAuth.ID == *id {
			return &appAuth, nil
		}
	}

	return nil, util.NewErrNotFound("application authentication")
}

func (m MockApplicationAuthenticationDao) Create(src *m.ApplicationAuthentication) error {
	return nil
}

func (m MockApplicationAuthenticationDao) Update(src *m.ApplicationAuthentication) error {
	panic("implement me")
}

func (m MockApplicationAuthenticationDao) Delete(id *int64) (*m.ApplicationAuthentication, error) {
	return m.GetById(id)
}

func (m MockApplicationAuthenticationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (m MockApplicationAuthenticationDao) ApplicationAuthenticationsByResource(_ string, _ []m.Application, _ []m.Authentication) ([]m.ApplicationAuthentication, error) {
	return m.ApplicationAuthentications, nil
}
