package mocks

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

var conf = config.Get()

type MockAuthenticationDao struct {
	Authentications []m.Authentication
}

func (mockAuthDao MockAuthenticationDao) List(_, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	count := int64(len(mockAuthDao.Authentications))
	return mockAuthDao.Authentications, count, nil
}

func (mockAuthDao MockAuthenticationDao) GetById(id string) (*m.Authentication, error) {
	for _, auth := range mockAuthDao.Authentications {
		// If secret store is database, we compare given ID with different field
		// than if secret store is vault
		if conf.SecretStore == "database" {
			if fmt.Sprintf("%d", auth.DbID) == id {
				return &auth, nil
			}
		} else {
			if auth.ID == id {
				return &auth, nil
			}
		}
	}

	return nil, util.NewErrNotFound("authentication")
}

func (mockAuthDao MockAuthenticationDao) ListForSource(sourceID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	sourceExists := false

	for _, src := range fixtures.TestSourceData {
		if src.ID == sourceID {
			sourceExists = true
		}
	}

	if !sourceExists {
		return nil, 0, util.NewErrNotFound("source")
	}

	out := make([]m.Authentication, 0)

	for _, auth := range mockAuthDao.Authentications {
		if auth.SourceID == sourceID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mockAuthDao MockAuthenticationDao) ListForApplication(appId int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	var appExists bool

	for _, app := range fixtures.TestApplicationData {
		if app.ID == appId {
			appExists = true
		}
	}

	if !appExists {
		return nil, 0, util.NewErrNotFound("application")
	}

	var out []m.Authentication

	for _, auth := range fixtures.TestAuthenticationData {
		if auth.ResourceType == "Application" && auth.ResourceID == appId {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mockAuthDao MockAuthenticationDao) ListForApplicationAuthentication(appAuthID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	var (
		appAuthExists bool
		authID        int64
	)

	for _, appAuth := range fixtures.TestApplicationAuthenticationData {
		if appAuth.ID == appAuthID {
			authID = appAuth.AuthenticationID
			appAuthExists = true

			break
		}
	}

	if !appAuthExists {
		return nil, 0, util.NewErrNotFound("application authentication")
	}

	auth, err := mockAuthDao.GetById(fmt.Sprintf("%d", authID))
	if err != nil {
		return nil, 0, err
	}

	authentications := []m.Authentication{*auth}

	return authentications, int64(1), nil
}

func (mockAuthDao MockAuthenticationDao) ListForEndpoint(endpointID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	endpointExists := false

	for _, e := range fixtures.TestEndpointData {
		if e.ID == endpointID {
			endpointExists = true
			break
		}
	}

	if !endpointExists {
		return nil, 0, util.NewErrNotFound("endpoint")
	}

	out := make([]m.Authentication, 0)

	for _, auth := range mockAuthDao.Authentications {
		if auth.ResourceType == "Endpoint" && auth.ResourceID == endpointID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mockAuthDao MockAuthenticationDao) Create(auth *m.Authentication) error {
	switch auth.ResourceType {
	case "Application", "Endpoint", "Source":
		return nil
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}
}

func (mockAuthDao MockAuthenticationDao) Update(auth *m.Authentication) error {
	if auth.ID == fixtures.TestAuthenticationData[0].ID {
		return nil
	}

	return util.NewErrNotFound("authentication")
}

func (mockAuthDao MockAuthenticationDao) Delete(id string) (*m.Authentication, error) {
	for _, auth := range mockAuthDao.Authentications {
		// If secret store is database, we compare given ID with different field
		// than if secret store is vault
		if conf.SecretStore == "database" {
			if fmt.Sprintf("%d", auth.DbID) == id {
				return &auth, nil
			}
		} else {
			if auth.ID == id {
				return &auth, nil
			}
		}
	}

	return nil, util.NewErrNotFound("authentication")
}

func (mockAuthDao MockAuthenticationDao) Tenant() *int64 {
	fakeTenantId := int64(12345)
	return &fakeTenantId
}

func (mockAuthDao MockAuthenticationDao) AuthenticationsByResource(_ *m.Authentication) ([]m.Authentication, error) {
	panic("implement me")
}

func (mockAuthDao MockAuthenticationDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	panic("implement me")
}

func (mockAuthDao MockAuthenticationDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	panic("implement me")
}

func (mockAuthDao MockAuthenticationDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	panic("implement me")
}

func (mockAuthDao MockAuthenticationDao) BulkCreate(_ *m.Authentication) error {
	panic("implement me")
}

func (mockAuthDao MockAuthenticationDao) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
	var authsList []m.Authentication

	for _, auth := range fixtures.TestAuthenticationData {
		for _, rid := range resourceIds {
			if auth.ResourceType == resourceType && auth.ResourceID == rid {
				authsList = append(authsList, auth)
			}
		}
	}

	if authsList == nil {
		return nil, nil
	} else {
		return authsList, nil
	}
}

// BulkDelete deletes all the authentications given as a list, and returns the ones that were deleted.
func (mockAuthDao MockAuthenticationDao) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	return authentications, nil
}
