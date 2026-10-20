package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// implement the auth dao but return errors everywhere that will bubble up to the end user.
type noSecretStoreAuthenticationDao struct{}

func (a *noSecretStoreAuthenticationDao) List(limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return nil, 0, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) GetById(id string) (*m.Authentication, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ListForSource(sourceID int64, limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return nil, 0, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ListForApplication(applicationID int64, limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return nil, 0, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ListForApplicationAuthentication(appAuthID int64, limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return nil, 0, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ListForEndpoint(endpointID int64, limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return nil, 0, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) Create(src *m.Authentication) error {
	return m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) BulkCreate(src *m.Authentication) error {
	return m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) Update(src *m.Authentication) error {
	return m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) Delete(id string) (*m.Authentication, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) Tenant() *int64 {
	return new(int64)
}

func (a *noSecretStoreAuthenticationDao) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ToEventJSON(resource util.Resource) ([]byte, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
	return nil, m.ErrBadSecretStore
}

func (a *noSecretStoreAuthenticationDao) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	return nil, m.ErrBadSecretStore
}

// implement the secrets dao interface, embedding the parent so we only have to implement the overridden names
type noSecretStoreSecretsDao struct {
	noSecretStoreAuthenticationDao
}

func (n *noSecretStoreSecretsDao) Delete(id *int64) error {
	return m.ErrBadSecretStore
}

func (n *noSecretStoreSecretsDao) NameExistsInCurrentTenant(name string) bool {
	return false
}

func (n *noSecretStoreSecretsDao) GetById(id *int64) (*m.Authentication, error) {
	return n.noSecretStoreAuthenticationDao.GetById("")
}

func (n *noSecretStoreSecretsDao) List(limit int, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	return n.noSecretStoreAuthenticationDao.List(limit, offset, filters)
}

func (n *noSecretStoreSecretsDao) Update(src *m.Authentication) error {
	return n.noSecretStoreAuthenticationDao.Update(src)
}
