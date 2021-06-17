package dao

import (
	m "github.com/lindgrenj6/sources-api-go/model"
)

type SourceDao interface {
	Count() (int, error)
	List(limit, offset int) ([]m.Source, error)
	GetById(id *int64) (*m.Source, error)
	Create(src *m.Source) (*int64, error)
	Update(src *m.Source) error
	Delete(id *int64) error
	Tenant() *int64
}
