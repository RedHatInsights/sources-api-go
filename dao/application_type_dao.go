package dao

import (
	"fmt"
	"strconv"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// GetApplicationTypeDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetApplicationTypeDao func(*int64) ApplicationTypeDao

// getDefaultApplicationAuthenticationDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultApplicationTypeDao(tenantId *int64) ApplicationTypeDao {
	return &applicationTypeDaoImpl{
		TenantID: tenantId,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetApplicationTypeDao = getDefaultApplicationTypeDao
}

type applicationTypeDaoImpl struct {
	TenantID *int64
}

func (a *applicationTypeDaoImpl) Exists(id *int64) (bool, error) {
	var exists bool

	result := DB.Debug().
		Select("1").
		Model(&m.ApplicationType{Id: *id}).
		First(&exists)

	return exists, result.Error
}

func (a *applicationTypeDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	appTypes := make([]m.ApplicationType, 0, limit)
	query := DB.Debug().Model(&m.ApplicationType{})

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

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

// this one breaks the pattern of just appending a single filter.
// it results in 2 queries instead of one but arguably is easier to understand.
func (a *applicationTypeDaoImpl) ListForSource(sourceID *int64, limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	exists, err := GetSourceDao(a.TenantID).Exists(sourceID)
	if !exists || err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	// pull the applications for the source
	applications, _, err := GetApplicationDao(a.TenantID).ListForSource(sourceID, 100, 0, []util.Filter{})
	if err != nil {
		return nil, 0, err
	}

	// convert the int foreign keys to strings
	applicationTypeIds := make([]string, len(applications))
	for i, app := range applications {
		applicationTypeIds[i] = strconv.FormatInt(app.ApplicationTypeID, 10)
	}

	// list with the filter
	return a.List(limit, offset, append(filters, util.Filter{Name: "id", Value: applicationTypeIds}))
}

func (a *applicationTypeDaoImpl) GetById(id *int64) (*m.ApplicationType, error) {
	appType := &m.ApplicationType{Id: *id}
	result := DB.Debug().First(appType)
	if result.Error != nil {
		return nil, util.NewErrNotFound("application type")
	}

	return appType, nil
}

func (a *applicationTypeDaoImpl) GetByName(name string) (*m.ApplicationType, error) {
	apptype := &m.ApplicationType{}
	result := DB.Debug().Where("name LIKE ?", "%"+name+"%").First(&apptype)

	return apptype, result.Error
}

func (a *applicationTypeDaoImpl) Create(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *applicationTypeDaoImpl) Update(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *applicationTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}

func (at *applicationTypeDaoImpl) ApplicationTypeCompatibleWithSource(typeId, sourceId int64) error {
	// Looks up the source ID and then compare's the source-type's name with the
	// application type's supported source types
	source := m.Source{ID: sourceId}
	result := DB.Debug().Preload("SourceType").Find(&source)
	if result.Error != nil {
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
	result := DB.Debug().
		Select("application_types.*").
		Joins("LEFT JOIN source_types ON source_types.id = ?", sourceTypeId).
		Where("application_types.id = ?", appTypeId).
		Where("application_types.supported_source_types::jsonb ? source_types.name").
		First(&m.ApplicationType{})

	return result.Error
}

func (at *applicationTypeDaoImpl) GetSuperKeyResultType(applicationTypeId int64, authType string) (string, error) {
	resultType := ""

	// selecting the authtype's supported authentication types for superkey via
	// jsonb query
	//
	// the short story is that we're pulling the `authType` key out of the
	// supportedAuthenticationTypes which is an array and then plucking index 0
	result := DB.Debug().
		Model(&m.ApplicationType{Id: applicationTypeId}).
		Select("application_types.supported_authentication_types::json -> ? ->> 0", authType).
		Scan(&resultType)

	if result.Error != nil {
		return "", result.Error
	}

	return resultType, result.Error
}
