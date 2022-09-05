package mocks

import (
	"fmt"
	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

var conf = config.Get()

type MockApplicationAuthenticationDao struct {
	ApplicationAuthentications []m.ApplicationAuthentication
}

type MockAuthenticationDao struct {
	Authentications []m.Authentication
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

func (m MockAuthenticationDao) List(limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	count := int64(len(m.Authentications))
	return m.Authentications, count, nil
}

func (m MockAuthenticationDao) GetById(id string) (*m.Authentication, error) {
	for _, auth := range m.Authentications {
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

func (mAuth MockAuthenticationDao) ListForSource(sourceID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
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

	for _, auth := range mAuth.Authentications {
		if auth.SourceID == sourceID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (mAuth MockAuthenticationDao) ListForApplication(appId int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
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

func (mAuthDao MockAuthenticationDao) ListForApplicationAuthentication(appAuthID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	var appAuthExists bool
	var authID int64

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

	auth, err := mAuthDao.GetById(fmt.Sprintf("%d", authID))
	if err != nil {
		return nil, 0, err
	}

	authentications := []m.Authentication{*auth}

	return authentications, int64(1), nil
}

func (mAuth MockAuthenticationDao) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
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

	for _, auth := range mAuth.Authentications {
		if auth.ResourceType == "Endpoint" && auth.ResourceID == endpointID {
			out = append(out, auth)
		}
	}

	return out, int64(len(out)), nil
}

func (m MockAuthenticationDao) Create(auth *m.Authentication) error {
	switch auth.ResourceType {
	case "Application", "Endpoint", "Source":
		return nil
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}
}

func (m MockAuthenticationDao) Update(auth *m.Authentication) error {
	if auth.ID == fixtures.TestAuthenticationData[0].ID {
		return nil
	}
	return util.NewErrNotFound("authentication")
}

func (m MockAuthenticationDao) Delete(id string) (*m.Authentication, error) {
	for _, auth := range m.Authentications {
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

func (m MockAuthenticationDao) Tenant() *int64 {
	fakeTenantId := int64(12345)
	return &fakeTenantId
}

func (m MockAuthenticationDao) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) ToEventJSON(resource util.Resource) ([]byte, error) {
	panic("implement me")
}

func (m MockAuthenticationDao) BulkCreate(auth *m.Authentication) error {
	panic("AAA")
}

func (mad MockAuthenticationDao) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
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
func (m MockAuthenticationDao) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	return authentications, nil
}
