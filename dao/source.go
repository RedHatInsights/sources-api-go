package dao

import (
	"context"
	"time"

	m "github.com/lindgrenj6/sources-api-go/model"
)

var ctx = context.Background()

func SourceList(limit, offset int) ([]m.Source, int, error) {
	var sources []m.Source
	err := DB.NewSelect().
		Model(&m.Source{}).
		Limit(limit).
		Offset(offset).
		Order("id").
		Scan(ctx, &sources)

	return sources, len(sources), err
}

func SourceGet(id int64) (*m.Source, error) {
	src := &m.Source{Id: id}
	err := DB.NewSelect().
		Model(src).
		WherePK().
		Scan(ctx, src)

	return src, err
}

func SourceCreate(s *m.Source) (int64, error) {
	now := time.Now()
	s.TimeStamps = m.TimeStamps{
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	s.TenantId = 1

	id := int64(0)
	_, err := DB.NewInsert().
		Model(s).
		Returning("id").
		Exec(ctx, &id)

	if err != nil {
		return 0, err
	}

	return id, err
}

func SourceUpdate(s *m.Source) error {
	now := time.Now()
	s.UpdatedAt = &now

	_, err := DB.NewUpdate().
		Model(s).
		WherePK().
		Exec(ctx)

	return err
}

func SourceDelete(id int64) error {
	_, err := DB.NewDelete().
		Model(&m.Source{Id: id}).
		WherePK().
		Exec(ctx)

	return err
}
