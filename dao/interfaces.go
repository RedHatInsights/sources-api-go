package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/hashicorp/vault/api"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type SourceDao interface {
	// List lists all the sources from a given tenant, which should be specified in the request.
	List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error)
	// ListInternal lists all the existing sources.
	ListInternal(limit, offset int, filters []util.Filter) ([]m.Source, int64, error)
	SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error)
	GetById(id *int64) (*m.Source, error)
	Create(src *m.Source) error
	Update(src *m.Source) error
	Delete(id *int64) (*m.Source, error)
	Tenant() *int64
	NameExistsInCurrentTenant(name string) bool
	GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error)
	// ListForRhcConnection gets all the sources that are related to a given rhcConnection id.
	ListForRhcConnection(rhcConnectionId *int64, limit, offset int, filters []util.Filter) ([]m.Source, int64, error)
	BulkMessage(resource util.Resource) (map[string]interface{}, error)
	FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error)
	ToEventJSON(resource util.Resource) ([]byte, error)
	// Pause pauses the given source and all its dependant applications.
	Pause(id int64) error
	// Unpause resumes the given source and all its dependant applications.
	Unpause(id int64) error
	IsSuperkey(id int64) bool
	// DeleteCascade deletes the source along with all its related sub resources. It returns all the deleted
	// sub resources and the source itself.
	DeleteCascade(sourceId int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error)
	// Exists returns true if the source exists.
	Exists(sourceId int64) (bool, error)
}

type ApplicationDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.Application, int64, error)
	SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Application, int64, error)
	GetById(id *int64) (*m.Application, error)
	Create(src *m.Application) error
	Update(src *m.Application) error
	Delete(id *int64) (*m.Application, error)
	Tenant() *int64
	BulkMessage(resource util.Resource) (map[string]interface{}, error)
	FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error)
	ToEventJSON(resource util.Resource) ([]byte, error)
	// Pause pauses the application.
	Pause(id int64) error
	// Unpause resumes the application.
	Unpause(id int64) error
	GetByIdWithPreload(id *int64, preloads ...string) (*m.Application, error)
	IsSuperkey(id int64) bool
	// DeleteCascade deletes the application along with all its related application authentications.
	DeleteCascade(applicationId int64) ([]m.ApplicationAuthentication, *m.Application, error)
	// Exists returns true if the application exists.
	Exists(applicationId int64) (bool, error)
}

type AuthenticationDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error)
	GetById(id string) (*m.Authentication, error)
	ListForSource(sourceID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error)
	ListForApplication(applicationID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error)
	ListForApplicationAuthentication(appAuthID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error)
	ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error)
	Create(src *m.Authentication) error
	BulkCreate(src *m.Authentication) error
	Update(src *m.Authentication) error
	Delete(id string) (*m.Authentication, error)
	Tenant() *int64
	AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error)
	BulkMessage(resource util.Resource) (map[string]interface{}, error)
	FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error)
	ToEventJSON(resource util.Resource) ([]byte, error)
	// ListIdsForResource fetches all the authentication IDs for the given resource. The rest of the fields will be
	// either nil or default values.
	ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error)
	// BulkDelete deletes all the authentications given as a list, and returns the ones that were deleted.
	BulkDelete(authentications []m.Authentication) ([]m.Authentication, error)
}

type ApplicationAuthenticationDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.ApplicationAuthentication, int64, error)
	GetById(id *int64) (*m.ApplicationAuthentication, error)
	Create(src *m.ApplicationAuthentication) error
	Update(src *m.ApplicationAuthentication) error
	Delete(id *int64) (*m.ApplicationAuthentication, error)
	Tenant() *int64
	ApplicationAuthenticationsByResource(resourceType string, applications []m.Application, authentications []m.Authentication) ([]m.ApplicationAuthentication, error)
}

type ApplicationTypeDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error)
	SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error)
	GetById(id *int64) (*m.ApplicationType, error)
	Create(src *m.ApplicationType) error
	Update(src *m.ApplicationType) error
	Delete(id *int64) error
	ApplicationTypeCompatibleWithSource(typeId, sourceId int64) error
	GetSuperKeyResultType(applicationTypeId int64, authType string) (string, error)
	ApplicationTypeCompatibleWithSourceType(appTypeId, sourceTypeId int64) error
	GetByName(name string) (*m.ApplicationType, error)
}

type EndpointDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error)
	SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error)
	GetById(id *int64) (*m.Endpoint, error)
	Create(src *m.Endpoint) error
	Update(src *m.Endpoint) error
	Delete(id *int64) (*m.Endpoint, error)
	Tenant() *int64
	// CanEndpointBeSetAsDefaultForSource checks if the endpoint can be set as default, by checking if the given source
	// id already has another endpoint marked as default.
	CanEndpointBeSetAsDefaultForSource(sourceId int64) bool
	// IsRoleUniqueForSource checks if the role is unique for the given source ID.
	IsRoleUniqueForSource(role string, sourceId int64) bool
	// SourceHasEndpoints returns true if the provided source has any associated endpoints.
	SourceHasEndpoints(sourceId int64) bool
	BulkMessage(resource util.Resource) (map[string]interface{}, error)
	FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error)
	ToEventJSON(resource util.Resource) ([]byte, error)
	// Exists returns true if the endpoint exists.
	Exists(endpointId int64) (bool, error)
}

type MetaDataDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.MetaData, int64, error)
	SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.MetaData, int64, error)
	GetById(id *int64) (*m.MetaData, error)
	GetSuperKeySteps(applicationTypeId int64) ([]m.MetaData, error)
	GetSuperKeyAccountNumber(applicationTypeId int64) (string, error)
	ApplicationOptedIntoRetry(applicationTypeId int64) (bool, error)
}

type SourceTypeDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.SourceType, int64, error)
	GetById(id *int64) (*m.SourceType, error)
	Create(src *m.SourceType) error
	Update(src *m.SourceType) error
	Delete(id *int64) error
	GetByName(name string) (*m.SourceType, error)
}

type VaultClient interface {
	Read(path string) (*api.Secret, error)
	List(path string) (*api.Secret, error)
	Write(path string, data map[string]interface{}) (*api.Secret, error)
	Delete(path string) (*api.Secret, error)
}

type RhcConnectionDao interface {
	List(limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error)
	GetById(id *int64) (*m.RhcConnection, error)
	Create(rhcConnection *m.RhcConnection) (*m.RhcConnection, error)
	Update(rhcConnection *m.RhcConnection) error
	Delete(id *int64) (*m.RhcConnection, error)
	// ListForSource gets all the related connections to the given source id.
	ListForSource(sourceId *int64, limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error)
}

type TenantDao interface {
	// GetOrCreateTenantID returns the ID of the tenant associated with the provided identity. It tries to fetch the
	// tenant by its OrgId, and if it is not present, by its EBS account number.
	GetOrCreateTenantID(identity *identity.Identity) (int64, error)
	// TenantByIdentity returns the tenant associated to the given identity. It tries to fetch the tenant by its OrgId,
	// and if it is not preset, by its EBS account number.
	TenantByIdentity(identity *identity.Identity) (*m.Tenant, error)
}
