package dao

import (
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type ApplicationTypeDaoImpl struct {
	TenantID *int64
}

func (a *ApplicationTypeDaoImpl) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []middleware.Filter) ([]m.ApplicationType, *int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	applicationTypes := make([]m.ApplicationType, 0, limit)

	applicationType, err := m.NewRelationObject(primaryCollection, *a.TenantID, DB.Debug())
	if err != nil {
		return nil, nil, err
	}
	query := applicationType.HasMany(&m.ApplicationType{}, DB.Debug())

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Model(&m.ApplicationType{}).Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&applicationTypes)

	return applicationTypes, &count, result.Error
}

func (a *ApplicationTypeDaoImpl) List(limit, offset int, filters []middleware.Filter) ([]m.ApplicationType, int64, error) {
	// allocating a slice of application types, initial length of
	// 0, size of limit (since we will not be returning more than that)
	apptypes := make([]m.ApplicationType, 0, limit)
	query := DB.Model(&m.ApplicationType{}).Debug()

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// getting the total count (filters included) for pagination
	count := int64(0)
	query.Count(&count)

	// limiting + running the actual query.
	result := query.Limit(limit).Offset(offset).Find(&apptypes)

	return apptypes, count, result.Error
}

func (a *ApplicationTypeDaoImpl) GetById(id *int64) (*m.ApplicationType, error) {
	apptype := &m.ApplicationType{Id: *id}
	result := DB.Debug().First(apptype)

	return apptype, result.Error
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
