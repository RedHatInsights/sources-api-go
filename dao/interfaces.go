package dao

import (
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type SourceDao interface {
	List(limit, offset int, filters []middleware.Filter) ([]m.Source, *int64, error)
	GetById(id *int64) (*m.Source, error)
	Create(src *m.Source) error
	Update(src *m.Source) error
	Delete(id *int64) error
	Tenant() *int64
}

type ApplicationTypeDao interface {
	List(limit, offset int, filters []middleware.Filter) ([]m.ApplicationType, *int64, error)
	GetById(id *int64) (*m.ApplicationType, error)
	Create(src *m.ApplicationType) error
	Update(src *m.ApplicationType) error
	Delete(id *int64) error
}

type SourceTypeDao interface {
	List(limit, offset int, filters []middleware.Filter) ([]m.SourceType, *int64, error)
	GetById(id *int64) (*m.SourceType, error)
	Create(src *m.SourceType) error
	Update(src *m.SourceType) error
	Delete(id *int64) error
}
