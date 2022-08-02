package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

type authenticationDaoDbImpl struct {
	TenantID *int64
	UserID   *int64
	ctx      context.Context
}

func (add *authenticationDaoDbImpl) getDb() *gorm.DB {
	if add.TenantID == nil {
		panic("nil tenant found in sourceDaoImpl DAO")
	}

	query := DB.Debug().WithContext(add.ctx)
	query = query.Where("tenant_id = ?", add.TenantID)

	if add.UserID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", add.UserID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	return query
}

func (add *authenticationDaoDbImpl) getDbWithModel() *gorm.DB {
	return add.getDb().Model(&m.Authentication{})
}

func (add *authenticationDaoDbImpl) List(limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	query := add.getDbWithModel()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	authentications := make([]m.Authentication, 0, limit)
	err = query.
		Limit(limit).
		Offset(offset).
		Find(&authentications).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return authentications, count, nil
}

func (add *authenticationDaoDbImpl) GetById(id string) (*m.Authentication, error) {
	var authentication m.Authentication

	err := DB.
		Debug().
		Model(&m.Authentication{}).
		Where("id = ?", id).
		Where("tenant_id = ?", add.TenantID).
		First(&authentication).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("authentication")
	}

	return &authentication, nil
}

func (add *authenticationDaoDbImpl) ListForSource(sourceID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	// Check that the source exists before continuing.
	var sourceExists bool
	err := DB.Debug().
		Model(&m.Source{}).
		Select(`1`).
		Where(`id = ?`, sourceID).
		Where(`tenant_id = ?`, add.TenantID).
		Scan(&sourceExists).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	if !sourceExists {
		return nil, 0, util.NewErrNotFound("source")
	}

	// List and count all the authentications from the given source.
	query := add.getDbWithModel()

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.
		Where("source_id = ?", sourceID).
		Count(&count)

	// limiting + running the actual query.
	authentications := make([]m.Authentication, 0, limit)
	err = query.
		Limit(limit).
		Offset(offset).
		Find(&authentications).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return authentications, count, nil
}

func (add *authenticationDaoDbImpl) ListForApplication(applicationID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	// Check that the application exists before continuing.
	var applicationExists bool
	err := DB.Debug().
		Model(&m.Application{}).
		Select(`1`).
		Where(`id = ?`, applicationID).
		Where(`tenant_id = ?`, add.TenantID).
		Scan(&applicationExists).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	if !applicationExists {
		return nil, 0, util.NewErrNotFound("application")
	}

	// List and count all the authentications from the given application.
	query := add.getDbWithModel()

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.
		Where("resource_id = ?", applicationID).
		Where("resource_type = 'Application'").
		Count(&count)

	// limiting + running the actual query.
	authentications := make([]m.Authentication, 0, limit)
	err = query.
		Where("resource_id = ?", applicationID).
		Where("resource_type = 'Application'").
		Limit(limit).
		Offset(offset).
		Find(&authentications).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return authentications, count, nil
}

func (add *authenticationDaoDbImpl) ListForApplicationAuthentication(appAuthID int64, _, _ int, _ []util.Filter) ([]m.Authentication, int64, error) {
	// Get application authentication
	var appAuth m.ApplicationAuthentication

	err := add.getDb().
		Where("id = ?", appAuthID).
		First(&appAuth).
		Error

	if err != nil {
		return nil, 0, util.NewErrNotFound("application authentication")
	}

	// Get authentication
	auth, err := add.GetById(fmt.Sprintf("%d", appAuth.AuthenticationID))
	if err != nil {
		return nil, 0, err
	}

	authentications := []m.Authentication{*auth}

	return authentications, int64(1), nil
}

func (add *authenticationDaoDbImpl) ListForEndpoint(endpointID int64, limit, offset int, filters []util.Filter) ([]m.Authentication, int64, error) {
	// Check that the endpoint exists before continuing.
	var endpointExists bool
	err := DB.Debug().
		Model(&m.Endpoint{}).
		Select(`1`).
		Where(`id = ?`, endpointID).
		Where(`tenant_id = ?`, add.TenantID).
		Scan(&endpointExists).
		Error

	if err != nil {
		return nil, 0, err
	}

	if !endpointExists {
		return nil, 0, util.NewErrNotFound("endpoint")
	}

	// List and count all the authentications from the given endpoint.
	query := add.getDbWithModel()

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.
		Where("resource_id = ?", endpointID).
		Where("resource_type = 'Endpoint'").
		Count(&count)

	// limiting + running the actual query.
	authentications := make([]m.Authentication, 0, limit)
	err = query.
		Where("resource_id = ?", endpointID).
		Where("resource_type = 'Endpoint'").
		Limit(limit).
		Offset(offset).
		Find(&authentications).
		Error

	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return authentications, count, nil
}

func (add *authenticationDaoDbImpl) Create(authentication *m.Authentication) error {
	query := DB.Debug().
		Where("tenant_id = ?", *add.TenantID)

	switch strings.ToLower(authentication.ResourceType) {
	case "application":
		var app m.Application
		err := query.
			Model(&m.Application{}).
			Where("id = ?", authentication.ResourceID).
			First(&app).
			Error

		if err != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", authentication.ResourceType, authentication.ResourceID)
		}

		authentication.SourceID = app.SourceID
	case "endpoint":
		var endpoint m.Endpoint
		err := query.
			Model(&m.Endpoint{}).
			Where("id = ?", authentication.ResourceID).
			First(&endpoint).
			Error

		if err != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", authentication.ResourceType, authentication.ResourceID)
		}

		authentication.SourceID = endpoint.SourceID
	case "source":
		var source m.Source
		err := query.
			Model(&m.Source{}).
			Where("id = ?", authentication.ResourceID).
			First(&source).
			Error

		if err != nil {
			return fmt.Errorf("resource not found with type [%v], id [%v]", authentication.ResourceType, authentication.ResourceID)
		}

		authentication.SourceID = authentication.ResourceID
	default:
		return fmt.Errorf("bad resource type, supported types are [Application, Endpoint, Source]")
	}

	authentication.TenantID = *add.TenantID // the TenantID gets injected in the middleware
	if authentication.Password != nil {
		encryptedValue, err := util.Encrypt(*authentication.Password)
		if err != nil {
			return err
		}

		authentication.Password = &encryptedValue
	}

	return DB.
		Debug().
		Create(authentication).
		Error
}

// BulkCreate method _without_ checking if the resource exists. Basically since this is the bulk-create method the
// resource doesn't exist yet and we know the source ID is set beforehand.
func (add *authenticationDaoDbImpl) BulkCreate(auth *m.Authentication) error {
	auth.TenantID = *add.TenantID // the TenantID gets injected in the middleware
	if auth.Password != nil {
		encryptedValue, err := util.Encrypt(*auth.Password)
		if err != nil {
			return err
		}

		auth.Password = &encryptedValue
	}

	return DB.Debug().Create(auth).Error
}

func (add *authenticationDaoDbImpl) Update(authentication *m.Authentication) error {
	return add.getDb().
		Updates(authentication).
		Error
}

func (add *authenticationDaoDbImpl) Delete(id string) (*m.Authentication, error) {
	var authentication m.Authentication

	err := add.getDb().
		Where("id = ?", id).
		First(&authentication).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("authentication")
	}

	err = add.getDb().
		Delete(&authentication).
		Error

	if err != nil {
		return nil, fmt.Errorf(`failed to delete authentication with id "%s"`, id)
	}

	return &authentication, nil
}

func (add *authenticationDaoDbImpl) Tenant() *int64 {
	return add.TenantID
}

func (add *authenticationDaoDbImpl) AuthenticationsByResource(authentication *m.Authentication) ([]m.Authentication, error) {
	var err error
	var resourceAuthentications []m.Authentication

	switch authentication.ResourceType {
	case "Source":
		resourceAuthentications, _, err = add.ListForSource(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	case "Endpoint":
		resourceAuthentications, _, err = add.ListForEndpoint(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	case "Application":
		resourceAuthentications, _, err = add.ListForApplication(authentication.ResourceID, DEFAULT_LIMIT, DEFAULT_OFFSET, nil)
	default:
		return nil, fmt.Errorf("unable to fetch authentications for %s", authentication.ResourceType)
	}

	if err != nil {
		return nil, err
	}

	return resourceAuthentications, nil
}

func (add *authenticationDaoDbImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	add.TenantID = &resource.TenantID
	authentication, err := add.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	return BulkMessageFromSource(&authentication.Source, authentication)
}

func (add *authenticationDaoDbImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	authentication, err := add.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	err = authentication.UpdateBy(updateAttributes)
	if err != nil {
		return nil, err
	}
	err = add.Update(authentication)
	if err != nil {
		return nil, err
	}

	sourceDao := GetSourceDao(&RequestParams{TenantID: add.TenantID})
	source, err := sourceDao.GetById(&authentication.SourceID)
	if err != nil {
		return nil, err
	}
	authentication.Source = *source

	return authentication, nil
}

func (add *authenticationDaoDbImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	add.TenantID = &resource.TenantID
	auth, err := add.GetById(resource.ResourceUID)
	if err != nil {
		return nil, err
	}

	auth.TenantID = resource.TenantID
	auth.Tenant = m.Tenant{ExternalTenant: resource.AccountNumber}
	authEvent := auth.ToEvent()
	data, err := json.Marshal(authEvent)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (add *authenticationDaoDbImpl) ListIdsForResource(resourceType string, resourceIds []int64) ([]m.Authentication, error) {
	var authentications []m.Authentication

	err := DB.
		Debug().
		Model(m.Authentication{}).
		Where("resource_type = ?", resourceType).
		Where("resource_id IN ?", resourceIds).
		Where("tenant_id = ?", add.TenantID).
		Find(&authentications).
		Error

	if err != nil {
		return nil, err
	}

	return authentications, err
}

func (add *authenticationDaoDbImpl) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	// Collect the ids to be able to find all the authentication objects for them.
	var authIds = make([]int64, len(authentications))
	for i, auth := range authentications {
		authIds[i] = auth.DbID
	}

	// The authentications are fetched with the "Tenant" table preloaded, so that all the objects are returned with the
	// "external_tenant" column populated. This is required to be able to raise events with the "tenant" key populated.
	//
	// The "len(objects) != 0" check to delete the resources is necessary to avoid Gorm issuing the "cannot batch
	// delete without a where condition" error. In theory this should not happen, since this function is expected to
	// be called with a "len(authentications) != 0" slice, but just to be safe...
	var dbAuths []m.Authentication
	err := DB.
		Debug().
		Preload("Tenant").
		Where("id IN ?", authIds).
		Where("tenant_id = ?", add.TenantID).
		Find(&dbAuths).
		Error

	if err != nil {
		return nil, err
	}

	if len(dbAuths) != 0 {
		err = DB.
			Debug().
			Where("tenant_id = ?", add.TenantID).
			Delete(&dbAuths).
			Error

		if err != nil {
			return nil, err
		}
	}

	return dbAuths, nil
}
