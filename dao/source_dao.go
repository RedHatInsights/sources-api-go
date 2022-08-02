package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetSourceDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetSourceDao func(*RequestParams) SourceDao

// getDefaultRhcConnectionDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultSourceDao(daoParams *RequestParams) SourceDao {
	var tenantID, userID *int64
	var ctx context.Context
	if daoParams != nil && daoParams.TenantID != nil {
		tenantID = daoParams.TenantID
		userID = daoParams.UserID
		ctx = daoParams.ctx
	}

	return &sourceDaoImpl{
		TenantID: tenantID,
		UserID:   userID,
		ctx:      ctx,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetSourceDao = getDefaultSourceDao
}

type sourceDaoImpl struct {
	TenantID *int64
	UserID   *int64
	ctx      context.Context
}

func (s *sourceDaoImpl) getDbWithTable(query *gorm.DB, table string) *gorm.DB {
	if s.TenantID == nil {
		panic("nil tenant found in sourceDaoImpl DAO")
	}

	var whereCondition string
	if table != "" {
		whereCondition = fmt.Sprintf("%s.", table)
	}

	query = query.Where(whereCondition+"tenant_id = ?", s.TenantID)

	return s.useUserForDB(query, table)
}

func (s *sourceDaoImpl) useUserForDB(query *gorm.DB, table string) *gorm.DB {
	var whereCondition string
	if table != "" {
		whereCondition = fmt.Sprintf("%s.", table)
	}

	if s.UserID != nil {
		condition := fmt.Sprintf("%[1]vuser_id IS NULL OR %[1]vuser_id = ?", whereCondition)
		query = query.Where(condition, s.UserID)
	} else {
		query = query.Where(whereCondition + "user_id IS NULL")
	}

	return query
}

func (s *sourceDaoImpl) getDb() *gorm.DB {
	return s.getDbWithTable(DB.Debug().WithContext(s.ctx), "")
}

func (s *sourceDaoImpl) getDbWithModel() *gorm.DB {
	return s.getDb().Model(&m.Source{})
}

func (s *sourceDaoImpl) useDbWithModel(query *gorm.DB) *gorm.DB {
	return s.getDbWithTable(query, "").Model(&m.Source{})
}

func (s *sourceDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	// allocating a slice of source types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	sources := make([]m.Source, 0, limit)

	relationObject, err := m.NewRelationObject(primaryCollection, *s.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
	}
	query := relationObject.HasMany(&m.Source{}, DB.Debug())

	query = s.getDbWithTable(query, "sources")

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.Source{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sources)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return sources, count, nil
}

func (s *sourceDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	sources := make([]m.Source, 0, limit)
	query := s.getDbWithModel()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&sources)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return sources, count, nil
}

func (s *sourceDaoImpl) ListInternal(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	query := DB.Debug().
		Model(&m.Source{}).
		Select(`sources.id, sources.availability_status, "Tenant".external_tenant, "Tenant".org_id`)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// Getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	sources := make([]m.Source, 0, limit)
	result := query.Joins("Tenant").Limit(limit).Offset(offset).Order("sources.id ASC").Find(&sources)

	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return sources, count, nil
}

func (s *sourceDaoImpl) GetById(id *int64) (*m.Source, error) {
	var src m.Source

	err := s.getDbWithModel().
		Where("id = ?", *id).
		First(&src).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("source")
	}

	return &src, nil
}

// GetByIdWithPreload searches for a source and preloads any specified relations.
func (s *sourceDaoImpl) GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error) {
	q := s.getDbWithModel().
		Where("id = ?", *id)

	for _, preload := range preloads {
		q = q.Preload(preload)
	}

	var src m.Source
	err := q.
		First(&src).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("source")
	}

	return &src, nil
}

func (s *sourceDaoImpl) Create(src *m.Source) error {
	src.TenantID = *s.TenantID // the TenantID gets injected in the middleware
	result := DB.Debug().Create(src)
	return result.Error
}

func (s *sourceDaoImpl) Update(src *m.Source) error {
	result := s.getDb().Updates(src)
	return result.Error
}

func (s *sourceDaoImpl) Delete(id *int64) (*m.Source, error) {
	var source m.Source

	result := s.getDb().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Delete(&source)

	if result.Error != nil {
		return nil, fmt.Errorf(`failed to source endpoint with id "%d": %s`, id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, util.NewErrNotFound("source")
	}

	return &source, nil
}

func (s *sourceDaoImpl) User() *int64 {
	return s.UserID
}

func (s *sourceDaoImpl) Tenant() *int64 {
	return s.TenantID
}

func (s *sourceDaoImpl) NameExistsInCurrentTenant(name string) bool {
	err := s.getDbWithModel().
		Where("name = ?", name).
		First(&m.Source{}).
		Error

	// If the name is found, GORM returns one row and no errors.
	return err == nil
}

func (s *sourceDaoImpl) IsSuperkey(id int64) bool {
	var valid bool
	result := s.getDbWithModel().
		Select("app_creation_workflow = ?", m.AccountAuth).
		Where("id = ?", id).
		First(&valid)

	if result.Error != nil {
		return false
	}

	return valid
}

func (s *sourceDaoImpl) BulkMessage(resource util.Resource) (map[string]interface{}, error) {
	var src m.Source

	err := s.getDbWithModel().
		Model(&m.Source{}).
		Where("id = ?", resource.ResourceID).
		Find(&src).
		Error

	if err != nil {
		return nil, err
	}

	authentication := &m.Authentication{ResourceID: src.ID, ResourceType: "Source"}

	return BulkMessageFromSource(&src, authentication)
}

func (s *sourceDaoImpl) FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error) {
	result := s.getDbWithModel().
		Where("id = ?", resource.ResourceID).
		Updates(updateAttributes)

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("source not found %v", resource)
	}

	source, err := s.GetById(&resource.ResourceID)
	if err != nil {
		return nil, err
	}

	return source, nil
}

func (s *sourceDaoImpl) FindWithTenant(id *int64) (*m.Source, error) {
	var src m.Source

	err := s.getDbWithModel().
		Where("id = ?", *id).
		Preload("Tenant").
		Find(&src).
		Error

	return &src, err
}

func (s *sourceDaoImpl) ToEventJSON(resource util.Resource) ([]byte, error) {
	src, err := s.FindWithTenant(&resource.ResourceID)
	if err != nil {
		return nil, err
	}

	data, errorJson := json.Marshal(src.ToEvent())
	if errorJson != nil {
		return nil, errorJson
	}

	return data, err
}

func (s *sourceDaoImpl) ListForRhcConnection(rhcConnectionId *int64, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	sources := make([]m.Source, 0)

	query := DB.
		Debug().
		Model(&m.Source{}).
		Joins(`INNER JOIN "source_rhc_connections" "sr" ON "sources"."id" = "sr"."source_id"`).
		Where(`"sr"."rhc_connection_id" = ?`, rhcConnectionId).
		Where(`"sr"."tenant_id" = ?`, s.TenantID)

	query = s.useUserForDB(query, "sources")

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// Getting the total count (filters included) for pagination.
	count := int64(0)
	query.Count(&count)

	// Run the actual query.
	err = query.Limit(limit).Offset(offset).Find(&sources).Error
	if err != nil {
		return nil, count, util.NewErrBadRequest(err)
	}

	return sources, count, nil
}

func (s *sourceDaoImpl) Pause(id int64) error {
	err := DB.Debug().Transaction(func(tx *gorm.DB) error {
		err := s.useDbWithModel(tx).
			Where("id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", time.Now()).
			Error

		if err != nil {
			return err
		}

		err = tx.Debug().
			Model(&m.Application{}).
			Where("source_id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", time.Now()).
			Error

		return err
	})

	return err
}

func (s *sourceDaoImpl) Unpause(id int64) error {
	err := DB.Debug().Transaction(func(tx *gorm.DB) error {
		err := s.useDbWithModel(tx).
			Where("id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", nil).
			Error

		if err != nil {
			return err
		}

		err = tx.Debug().
			Model(&m.Application{}).
			Where("source_id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", nil).
			Error

		return err
	})

	return err
}

func (s *sourceDaoImpl) DeleteCascade(sourceId int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error) {
	var applicationAuthentications []m.ApplicationAuthentication
	var applications []m.Application
	var endpoints []m.Endpoint
	var rhcConnections []m.RhcConnection
	var source *m.Source

	// The different items are fetched with the "Tenant" table preloaded, so that all the objects are returned with the
	// "external_tenant" column populated. This is required to be able to raise events with the "tenant" key populated.
	//
	// The "len(objects) != 0" check to delete the resources is necessary to avoid Gorm issuing the "cannot batch
	// delete without a where condition" error, since there might be times when the resources don't have any related
	// sub resources.
	err := DB.
		Debug().
		Transaction(func(tx *gorm.DB) error {
			// Fetch and delete the application authentications.
			err := tx.
				Model(&m.ApplicationAuthentication{}).
				Preload("Tenant").
				Joins(`INNER JOIN "applications" ON "application_authentications"."application_id" = "applications"."id"`).
				Where(`"applications"."source_id" = ?`, sourceId).
				Where(`"applications"."tenant_id" = ?`, s.TenantID).
				Find(&applicationAuthentications).
				Error

			if err != nil {
				return err
			}

			if len(applicationAuthentications) != 0 {
				err = tx.
					Delete(&applicationAuthentications).
					Error

				if err != nil {
					return err
				}
			}

			// Fetch and delete the applications.
			err = tx.
				Model(&m.Application{}).
				Preload("Tenant").
				Where("source_id = ?", sourceId).
				Where("tenant_id = ?", s.TenantID).
				Find(&applications).
				Error

			if err != nil {
				return err
			}

			if len(applications) != 0 {
				err = tx.
					Delete(&applications).
					Error

				if err != nil {
					return err
				}
			}

			// Fetch and delete the endpoints.
			err = tx.
				Model(m.Endpoint{}).
				Preload("Tenant").
				Where("source_id = ?", sourceId).
				Where("tenant_id = ?", s.TenantID).
				Find(&endpoints).
				Error

			if err != nil {
				return err
			}

			if len(endpoints) != 0 {
				err = tx.
					Delete(&endpoints).
					Error

				if err != nil {
					return err
				}
			}

			// Fetch and delete the rhcConnections.
			err = tx.
				Model(&m.RhcConnection{}).
				Joins(`INNER JOIN "source_rhc_connections" ON "source_rhc_connections"."rhc_connection_id" = "rhc_connections"."id"`).
				Where(`"source_rhc_connections"."source_id" = ?`, sourceId).
				Where(`"source_rhc_connections"."tenant_id" = ?`, s.TenantID).
				Find(&rhcConnections).
				Error

			if err != nil {
				return err
			}

			if len(rhcConnections) != 0 {
				err = tx.
					Delete(&rhcConnections).
					Error

				if err != nil {
					return err
				}
			}

			// Fetch and delete the source itself.
			err = tx.
				Preload("Tenant").
				Where("id = ?", sourceId).
				Where("tenant_id = ?", s.TenantID).
				Find(&source).
				Error

			if err != nil {
				return err
			}

			if source != nil {
				err = tx.
					Delete(&source).
					Error
			}

			return err
		})

	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return applicationAuthentications, applications, endpoints, rhcConnections, source, nil
}

func (s *sourceDaoImpl) Exists(sourceId int64) (bool, error) {
	var sourceExists bool

	err := s.getDbWithModel().
		Select("1").
		Where("id = ?", sourceId).
		Scan(&sourceExists).
		Error

	if err != nil {
		return false, err
	}

	return sourceExists, nil
}
