package dao

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetSourceDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetSourceDao func(*RequestParams) SourceDao

// getDefaultRhcConnectionDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultSourceDao(daoParams *RequestParams) SourceDao {
	return &sourceDaoImpl{RequestParams: daoParams}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetSourceDao = getDefaultSourceDao
}

type sourceDaoImpl struct {
	*RequestParams
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
	query := s.getDbWithTable(DB.Debug(), "sources").Model(&m.Source{})

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

func (s *sourceDaoImpl) ListInternal(limit, offset int, filters []util.Filter, skipEmptySources bool) ([]m.Source, int64, error) {
	query := DB.Debug().
		Model(&m.Source{}).
		Select(`sources.id, sources.availability_status, "Tenant".external_tenant, "Tenant".org_id`)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// The "Tenant" table must be joined here as otherwise joining it after the "group" and "having" statements does
	// not join the table for some reason.
	query.Joins("Tenant")

	// When told to do so, skip all the sources that don't have any associated applications or RHC Connections with
	// them. Useful for when the Sources Monitor requests all the sources to then perform availability check requests
	// with them. With the conditions below, we can ensure that only the Sources that require those availability checks
	// are checked.
	//
	// Per https://issues.redhat.com/browse/RHCLOUD-38735 we consider a source with just "Cost Management" applications
	// as not having any applications at all.
	if skipEmptySources {
		query.Group("sources.id")

		// These "groups" are required since Gorm appends more things to the "SELECT" query.
		query.Group(`"Tenant".id`)
		query.Group(`"Tenant".external_tenant`)
		query.Group(`"Tenant".org_id`)

		query.Having(`
			(
				SELECT
					COUNT(applications.*)
				FROM
					applications
				INNER JOIN
					application_types ON application_types.id = applications.application_type_id
				WHERE
					applications.source_id = sources.id
				AND
					application_types."name" != '/insights/platform/cost-management'
			) > 0
			OR
			(
				SELECT
					COUNT(source_rhc_connections.*)
				FROM
					source_rhc_connections
				WHERE
					source_id = sources.id
			) > 0
		`)
	}

	// Getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	sources := make([]m.Source, 0, limit)
	result := query.Limit(limit).Offset(offset).Order("sources.id ASC").Find(&sources)

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

	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID}).Errorf(`Unable to create source: %s`, result.Error)

		return result.Error
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": src.ID}).Info("Source created")

		return nil
	}
}

func (s *sourceDaoImpl) Update(src *m.Source) error {
	result := s.getDb().Omit(clause.Associations).Updates(src)

	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": s.TenantID, "source_id": src.ID}).Errorf(`Unable to update source: %s`, result.Error)

		return result.Error
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": s.TenantID, "source_id": src.ID}).Info("Source updated")

		return nil
	}
}

func (s *sourceDaoImpl) Delete(id *int64) (*m.Source, error) {
	var source m.Source

	result := s.getDb().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Delete(&source)

	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Errorf(`Unable to delete source: %s`, result.Error)

		return nil, fmt.Errorf(`failed to source endpoint with id "%d": %s`, id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, util.NewErrNotFound("source")
	}

	logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Info("Source deleted")

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
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Errorf(`Unable to pause source: %s`, err)

			return err
		}

		err = s.getDbWithTable(tx.Debug(), "").
			Model(&m.Application{}).
			Where("source_id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", time.Now()).
			Error
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Errorf(`Unable to pause source's applications': %s`, err)

			return err
		}

		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Info("Source and its applications paused")

		return nil
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
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Errorf(`Unable to resume source: %s`, err)

			return err
		}

		err = s.getDbWithTable(tx.Debug(), "").
			Model(&m.Application{}).
			Where("source_id = ?", id).
			Where("tenant_id = ?", s.TenantID).
			Update("paused_at", nil).
			Error
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Errorf(`Unable to resume source's applications': %s`, err)

			return err
		}

		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": id}).Info("Source and its applications resumed")

		return nil
	})

	return err
}

func (s *sourceDaoImpl) DeleteCascade(sourceId int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error) {
	var (
		applicationAuthentications []m.ApplicationAuthentication
		applications               []m.Application
		endpoints                  []m.Endpoint
		rhcConnections             []m.RhcConnection
		source                     *m.Source
	)

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
					logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Errorf("Unable to cascade delete source: unable to delete application authentications: %s", err)

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
					logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Errorf("Unable to cascade delete source: unable to delete applications: %s", err)

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
					logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Errorf("Unable to cascade delete source: unable to delete endpoints: %s", err)

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
					logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Errorf("Unable to cascade delete source: unable to delete RHC connections: %s", err)

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

			if err != nil {
				logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Errorf("Unable to cascade delete source: %s", err)

				return err
			} else {
				return nil
			}
		})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Log all the changes for observability, traceability and debugging purposes.
	for _, appAuth := range applicationAuthentications {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId, "application_authentication_id": appAuth.ID}).Info("Application authentication deleted")
	}

	for _, app := range applications {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId, "application_id": app.ID}).Info("Application deleted")
	}

	for _, endpoint := range endpoints {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId, "endpoint_id": endpoint.ID}).Info("Endpoint deleted")
	}

	for _, rhcConnection := range rhcConnections {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId, "rhc_connection_id": rhcConnection.ID}).Info("RHC connection deleted")
	}

	logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": sourceId}).Info("Source deleted")

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
