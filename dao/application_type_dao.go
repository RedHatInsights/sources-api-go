package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type ApplicationTypeDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationTypeDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	applicationTypes := make([]m.ApplicationType, 0, limit)

	applicationType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, 0, util.NewErrNotFound("source")
	}

	query := applicationType.HasMany(&m.ApplicationType{}, DB.Debug())

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.ApplicationType{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&applicationTypes)

	return applicationTypes, count, result.Error
}

func (a *ApplicationTypeDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	appTypes := make([]m.ApplicationType, 0, limit)
	query := DB.Model(&m.ApplicationType{}).Debug()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&appTypes)

	return appTypes, count, result.Error
}

func (a *ApplicationTypeDaoImpl) GetById(id *int64) (*m.ApplicationType, error) {
	appType := &m.ApplicationType{Id: *id}
	result := DB.Debug().First(appType)
	if result.Error != nil {
		return nil, util.NewErrNotFound("application type")
	}

	return appType, nil
}

func (a *ApplicationTypeDaoImpl) Create(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *ApplicationTypeDaoImpl) Update(_ *m.ApplicationType) error {
	panic("not needed (yet) due to seeding.")
}

func (a *ApplicationTypeDaoImpl) Delete(_ *int64) error {
	panic("not needed (yet) due to seeding.")
}

func (at *ApplicationTypeDaoImpl) ApplicationTypeCompatibleWithSource(typeId, sourceId int64) error {
	source := m.Source{ID: sourceId}
	result := DB.Preload("SourceType").Find(&source)
	if result.Error != nil {
		return fmt.Errorf("source not found")
	}

	// searching for the application type that has the source type's name in its
	// supported source types column.
	result = DB.First(
		&m.ApplicationType{Id: typeId},
		datatypes.JSONQuery("supported_source_types").HasKey(source.SourceType.Name),
	)

	return result.Error
}
