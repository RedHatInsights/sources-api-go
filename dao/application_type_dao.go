package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

// GetApplicationTypeDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetApplicationTypeDao func(*int64) ApplicationTypeDao

// getDefaultApplicationAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultApplicationTypeDao(tenantId *int64) ApplicationTypeDao {
	return &applicationTypeDaoImpl{
		SourcesConfig: config.Get(),
		TenantID:      tenantId,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetApplicationTypeDao = getDefaultApplicationTypeDao
}

type applicationTypeDaoImpl struct {
	SourcesConfig *config.SourcesApiConfig
	TenantID      *int64
}

func (at *applicationTypeDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	applicationTypes := make([]m.ApplicationType, 0, limit)

	relationObject, err := m.NewRelationObject(primaryCollection, *at.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, err
	}

	query := relationObject.HasMany(&m.ApplicationType{}, DB.Debug())

	query, err = applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err.Error())
	}

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.ApplicationType{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&applicationTypes)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return applicationTypes, count, nil
}

func (at *applicationTypeDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	appTypes := make([]m.ApplicationType, 0, limit)
	query := DB.Debug().Model(&m.ApplicationType{})

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&appTypes)
	if result.Error != nil {
		return nil, 0, util.NewErrBadRequest(result.Error)
	}

	return appTypes, count, nil
}

func (at *applicationTypeDaoImpl) GetById(id *int64) (*m.ApplicationType, error) {
	var appType m.ApplicationType

	query := DB.Debug().
		Model(&m.ApplicationType{}).
		Where("id = ?", *id)

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	if query.First(&appType).Error != nil {
		return nil, util.NewErrNotFound("application type")
	}

	return &appType, nil
}

func (at *applicationTypeDaoImpl) GetByName(name string) (*m.ApplicationType, error) {
	appTypes := make([]m.ApplicationType, 0)
	query := DB.Debug().
		Where("name LIKE ?", "%"+name+"%")

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	// Run the query.
	result := query.Find(&appTypes)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected > int64(1) {
		return nil, util.NewErrBadRequest("Found more than one of the same application type name")
	} else if result.RowsAffected == int64(0) {
		return nil, util.NewErrNotFound("application type")
	}

	return &appTypes[0], nil
}

func (at *applicationTypeDaoImpl) Create(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (at *applicationTypeDaoImpl) Update(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (at *applicationTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}

func (at *applicationTypeDaoImpl) ApplicationTypeCompatibleWithSource(typeId, sourceId int64) error {
	// Looks up the source ID and then compare's the source-type's name with the
	// application type's supported source types
	var source m.Source

	err := DB.Debug().
		Model(&m.Source{}).
		Where("id = ?", sourceId).
		Preload("SourceType").
		Find(&source).
		Error
	if err != nil {
		return err
	}

	if source.ID == 0 {
		return fmt.Errorf("source not found")
	}

	return at.ApplicationTypeCompatibleWithSourceType(typeId, source.SourceType.Id)
}

func (at *applicationTypeDaoImpl) ApplicationTypeCompatibleWithSourceType(appTypeId, sourceTypeId int64) error {
	// searching for the application type that has the source type's name in its
	// supported source types column.
	//
	// initially i tried to use the
	// datatypes.JsonQuery("application_types.supported_source_types") but that
	// doesn't work when we're specifying something joined in, in this case
	// "source_types.name"
	query := DB.Debug().
		Select("application_types.*").
		Joins("LEFT JOIN source_types ON source_types.id = ?", sourceTypeId).
		Where("application_types.id = ?", appTypeId).
		Where("application_types.supported_source_types::jsonb ? source_types.name")

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	return query.First(&m.ApplicationType{}).Error
}

func (at *applicationTypeDaoImpl) GetSuperKeyResultType(applicationTypeId int64, authType string) (string, error) {
	resultType := ""

	// selecting the authtype's supported authentication types for superkey via
	// jsonb query
	//
	// the short story is that we're pulling the `authType` key out of the
	// supportedAuthenticationTypes which is an array and then plucking index 0
	query := DB.Debug().
		Model(&m.ApplicationType{Id: applicationTypeId}).
		Select("application_types.supported_authentication_types::json -> ? ->> 0", authType)

	// Apply the disabled application types' restriction.
	at.applyDisabledApplicationTypesRestriction(query)

	// Run the query.
	err := query.Scan(&resultType).Error
	if err != nil {
		return "", err
	}

	return resultType, nil
}

// applyDisabledApplicationTypesRestriction applies a condition to the given
// query to avoid fetching the disabled application types from the database.
func (at *applicationTypeDaoImpl) applyDisabledApplicationTypesRestriction(query *gorm.DB) {
	if len(at.SourcesConfig.DisabledApplicationTypes) > 0 {
		query.Where("application_types.name NOT IN ?", at.SourcesConfig.DisabledApplicationTypes)
	}
}
