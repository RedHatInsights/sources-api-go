package dao

import (
	"context"
	"time"

	m "github.com/lindgrenj6/sources-api-go/model"
)

var ctx = context.Background()

type SourceDaoImpl struct {
	TenantID *int64
}

func (s *SourceDaoImpl) Tenant() *int64 {
	return s.TenantID
}

func (s *SourceDaoImpl) Count() (int, error) {
	count := 0
	err := DB.NewSelect().
		ColumnExpr("count(*)").
		Model(&m.Source{}).
		Where("tenant_id = ?", s.TenantID).
		Scan(ctx, &count)

	return count, err
}

func (s *SourceDaoImpl) List(limit, offset int) ([]m.Source, error) {
	sources := make([]m.Source, 0, limit)
	err := DB.NewSelect().
		Model(&m.Source{}).
		Where("tenant_id = ?", s.TenantID).
		Order("created_at asc").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &sources)

	return sources, err
}

func (s *SourceDaoImpl) GetById(id *int64) (*m.Source, error) {
	src := &m.Source{Id: *id}
	err := DB.NewSelect().
		Model(src).
		WherePK().
		Where("tenant_id = ?", s.TenantID).
		Scan(ctx, src)

	return src, err
}

func (s *SourceDaoImpl) Create(src *m.Source) (*int64, error) {
	now := time.Now()
	src.TimeStamps = m.TimeStamps{
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	id := int64(0)
	_, err := DB.NewInsert().
		Model(src).
		Returning("id").
		Exec(ctx, &id)

	if err != nil {
		return nil, err
	}

	return &id, err
}

func (s *SourceDaoImpl) Update(src *m.Source) error {
	now := time.Now()
	src.UpdatedAt = &now

	_, err := DB.NewUpdate().
		Model(src).
		WherePK().
		Where("tenant_id = ?", s.TenantID).
		Exec(ctx)

	return err
}

func (s *SourceDaoImpl) Delete(id *int64) error {
	var count int64
	_, err := DB.NewDelete().
		Model(&m.Source{Id: *id}).
		WherePK().
		Where("tenant_id = ?", s.TenantID).
		Returning("id").
		Exec(ctx, &count)

	if count != *id {
		return err
	} else {
		return nil
	}
}
