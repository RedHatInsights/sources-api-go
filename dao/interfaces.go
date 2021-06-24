package dao

import (
	"github.com/lindgrenj6/sources-api-go/middleware"
	m "github.com/lindgrenj6/sources-api-go/model"
)

type SourceDao interface {
	List(limit, offset int, filters []middleware.Filter) ([]m.Source, *int64, error)
	GetById(id *int64) (*m.Source, error)
	Create(src *m.Source) error
	Update(src *m.Source) error
	Delete(id *int64) error
	Tenant() *int64
}
