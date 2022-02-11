package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RhcConnectionDaoImpl struct{}

func (s *RhcConnectionDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error) {
	rhcConnections := make([]m.RhcConnection, 0, limit)
	query := DB.
		Debug().
		Model(&m.RhcConnection{}).
		Limit(limit).
		Offset(offset)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// Getting the total count (filters included) for pagination.
	count := int64(0)
	query.Count(&count)

	// Run the actual query.
	result := query.Find(&rhcConnections)

	return rhcConnections, count, result.Error
}

func (s *RhcConnectionDaoImpl) GetById(id *int64) (*m.RhcConnection, error) {
	rhcConnection := m.RhcConnection{
		ID: *id,
	}

	err := DB.
		Debug().
		Model(&m.RhcConnection{}).
		First(&rhcConnection).
		Error

	if err != nil {
		return nil, util.NewErrNotFound("rhcConnection")
	}

	return &rhcConnection, nil
}

func (s *RhcConnectionDaoImpl) Create(rhcConnection *m.RhcConnection) (*m.RhcConnection, error) {
	// If the source doesn't exist we cannot create the RhcConnection, since it needs to be linked to at least one
	// source.
	var sourceExists bool
	err := DB.Debug().
		Model(&m.Source{}).
		Select(`1`).
		Where(`id = ?`, rhcConnection.Sources[0].ID).
		Scan(&sourceExists).
		Error

	// Something went wrong with the query
	if err != nil {
		return nil, err
	}

	if !sourceExists {
		return nil, util.NewErrNotFound("source")
	}

	// Check if there's an already existing connection. If there is, we assume that the user wants to link the existing
	// connection to a different source.
	var connectionId int64
	err = DB.Debug().
		Model(&m.RhcConnection{}).
		Select(`id`).
		Where(`rhc_id = ?`, rhcConnection.RhcId).
		Scan(&connectionId).
		Error

	if err != nil {
		return nil, err
	}

	rhcConnection.ID = connectionId

	err = DB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Is it a new connection or is it just an association?
		if rhcConnection.ID == 0 {
			err = tx.Debug().
				Omit(clause.Associations).
				Create(&rhcConnection).
				Error
		}

		if err != nil {
			return err
		}

		// Try to insert an association. If it exists the database will complain.
		association := m.SourceRhcConnection{
			SourceId:        rhcConnection.Sources[0].ID,
			RhcConnectionId: rhcConnection.ID,
		}

		err = tx.Debug().
			Create(&association).
			Error

		if err != nil {
			return fmt.Errorf("cannot link red hat connection to source: %w", err)
		}

		return nil
	})

	return rhcConnection, err
}

func (s *RhcConnectionDaoImpl) Update(rhcConnection *m.RhcConnection) error {
	err := DB.Debug().
		Updates(rhcConnection).
		Error
	return err
}

func (s *RhcConnectionDaoImpl) Delete(id *int64) error {
	rhcConnection := &m.RhcConnection{ID: *id}

	err := DB.Debug().
		Where("id = ?", id).
		First(&rhcConnection).
		Error

	if err != nil {
		return util.NewErrNotFound("rhcConnection")
	}

	err = DB.Debug().Transaction(func(tx *gorm.DB) error {
		err := tx.
			Debug().
			Where(`id = ?`, *id).
			Delete(&m.RhcConnection{}).
			Error

		if err != nil {
			return err
		}

		err = tx.
			Debug().
			Where(`rhc_connection_id = ?`, *id).
			Delete(&m.SourceRhcConnection{}).
			Error

		return err
	})

	return err
}

func (s *RhcConnectionDaoImpl) GetRelatedSourcesToId(rhcConnectionId *int64, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	sources := make([]m.Source, 0)

	query := DB.Debug().
		Model(&m.Source{}).
		Joins(`INNER JOIN "source_rhc_connections" "sr" ON "sources"."id" = "sr"."source_id"`).
		Where(`"sr"."rhc_connection_id" = ?`, rhcConnectionId).
		Limit(limit).
		Offset(offset)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, err
	}

	// Getting the total count (filters included) for pagination.
	count := int64(0)
	query.Count(&count)

	// Run the actual query.
	err = query.Find(&sources).Error

	return sources, count, err

}
