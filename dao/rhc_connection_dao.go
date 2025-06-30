package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao/mappers"
	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetRhcConnectionDao is a function definition that can be replaced in runtime in case some other DAO provider is
// needed.
var GetRhcConnectionDao func(*RequestParams) RhcConnectionDao

// getDefaultRhcConnectionDao gets the default DAO implementation which will have the given tenant ID.
func getDefaultRhcConnectionDao(params *RequestParams) RhcConnectionDao {
	return &rhcConnectionDaoImpl{
		TenantID: params.TenantID,
		ctx:      params.ctx,
	}
}

// init sets the default DAO implementation so that other packages can request it easily.
func init() {
	GetRhcConnectionDao = getDefaultRhcConnectionDao
}

type rhcConnectionDaoImpl struct {
	TenantID *int64
	ctx      context.Context
}

func (s *rhcConnectionDaoImpl) getDb() *gorm.DB {
	return DB.Debug().WithContext(s.ctx)
}

func (s *rhcConnectionDaoImpl) List(limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error) {
	query := s.getDb().
		Model(&m.RhcConnection{}).
		Select(`"rhc_connections".*, STRING_AGG(CAST ("jt"."source_id" AS TEXT), ',') AS "source_ids"`).
		Joins(`INNER JOIN "source_rhc_connections" AS "jt" ON "rhc_connections"."id" = "jt"."rhc_connection_id"`).
		Where(`"jt"."tenant_id" = ?`, s.TenantID).
		Group(`"rhc_connections"."id"`)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// Getting the total count (filters included) for pagination.
	count := int64(0)
	query.Count(&count)

	// Run the actual query.
	result, err := query.Limit(limit).Offset(offset).Rows()
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// We call next as otherwise "ScanRows" complains, but since we're going to map the results to an array of
	// map[string]interface{}, "ScanRows" will already scan every row into that array, thus freeing us from calling
	// result.Next() again.
	if !result.Next() {
		return []m.RhcConnection{}, count, nil
	}

	// Loop through the rows to map both the connection and its related sources.
	var rows []map[string]interface{}

	err = DB.ScanRows(result, &rows)
	if err != nil {
		return nil, 0, err
	}

	rhcConnections := make([]m.RhcConnection, 0)

	for _, row := range rows {
		rhcConnection, err := mappers.MapRowToRhcConnection(row)
		if err != nil {
			return nil, 0, err
		}

		rhcConnections = append(rhcConnections, *rhcConnection)
	}

	err = result.Close()
	if err != nil {
		return nil, 0, err
	}

	return rhcConnections, count, nil
}

func (s *rhcConnectionDaoImpl) GetById(id *int64) (*m.RhcConnection, error) {
	query := s.getDb().
		Model(&m.RhcConnection{}).
		Select(`"rhc_connections".*, STRING_AGG(CAST ("jt"."source_id" AS TEXT), ',') AS "source_ids"`).
		Joins(`INNER JOIN "source_rhc_connections" AS "jt" ON "rhc_connections"."id" = "jt"."rhc_connection_id"`).
		Where(`"rhc_connections"."id" = ?`, id).
		Where(`"jt"."tenant_id" = ?`, s.TenantID).
		Group(`"rhc_connections"."id"`)

	// Run the actual query.
	result, err := query.Rows()
	if err != nil {
		return nil, err
	}

	// We call next as otherwise "ScanRows" complains, but since we're going to map the results to an array of
	// map[string]interface{}, "ScanRows" will already scan every row into that array, thus freeing us from calling
	// result.Next() again.
	if !result.Next() {
		return nil, util.NewErrNotFound("rhcConnection")
	}

	// Loop through the rows to map both the connection and its related sources.
	var rows []map[string]interface{}

	err = DB.ScanRows(result, &rows)
	if err != nil {
		return nil, err
	}

	err = result.Close()
	if err != nil {
		return nil, err
	}

	if len(rows) != 1 {
		return nil, errors.New("unexpected number of results")
	}

	rhcConnection, err := mappers.MapRowToRhcConnection(rows[0])
	if err != nil {
		return nil, err
	}

	return rhcConnection, nil
}

func (s *rhcConnectionDaoImpl) Create(rhcConnection *m.RhcConnection) (*m.RhcConnection, error) {
	// If the source doesn't exist we cannot create the RhcConnection, since it needs to be linked to at least one
	// source.
	var sourceExists bool

	err := s.getDb().
		Model(&m.Source{}).
		Select(`1`).
		Where(`id = ?`, rhcConnection.Sources[0].ID).
		Where(`tenant_id = ?`, s.TenantID).
		Scan(&sourceExists).
		Error

	// Something went wrong with the query
	if err != nil {
		return nil, err
	}

	if !sourceExists {
		return nil, util.NewErrNotFound("source")
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Debug().
			Where(`rhc_id = ?`, rhcConnection.RhcId).
			Omit(clause.Associations).
			FirstOrCreate(&rhcConnection).
			Error
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": rhcConnection.Sources[0].ID}).Errorf("Unable to create RHC connection: %s", err)

			return err
		}

		// Try to insert an sourceRhcConnection, which is just the relation between a rhcConnection and a source.
		sourceRhcConnection := m.SourceRhcConnection{
			SourceId:        rhcConnection.Sources[0].ID,
			RhcConnectionId: rhcConnection.ID,
			TenantId:        *s.TenantID,
		}

		// Check if it exists first.
		var relationExists bool

		err = tx.Debug().
			Model(&m.SourceRhcConnection{}).
			Select(`1`).
			Where(`source_id = ?`, sourceRhcConnection.SourceId).
			Where(`rhc_connection_id = ?`, sourceRhcConnection.RhcConnectionId).
			Where(`tenant_id = ?`, sourceRhcConnection.TenantId).
			Scan(&relationExists).
			Error
		if err != nil {
			return err
		}

		// If it exists, we let the client know. If it doesn't, we attempt to create it.
		if relationExists {
			return util.NewErrBadRequest("connection already exists")
		}

		err = tx.
			Debug().
			Create(&sourceRhcConnection).
			Error
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": rhcConnection.Sources[0].ID}).Errorf("Unable to create RHC connection association: %s", err)

			return err
		}

		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_id": rhcConnection.Sources[0].ID, "rhc_connection_id": rhcConnection.ID}).Info("RHC Connection created")

		return nil
	})

	return rhcConnection, err
}

func (s *rhcConnectionDaoImpl) Update(rhcConnection *m.RhcConnection) error {
	err := s.getDb().
		// We need to use the "Omit" clause since otherwise Gorm tries to create the associate source for the
		// connection as well.
		Omit(clause.Associations).
		Select("*").
		Updates(rhcConnection).
		Error
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_ids": rhcConnection.Sources, "rhc_connection_id": rhcConnection.ID}).Errorf("Unable to update RHC connection: %s", err)

		return err
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_ids": rhcConnection.Sources, "rhc_connection_id": rhcConnection.ID}).Info("RHC connection updated")

		return nil
	}
}

func (s *rhcConnectionDaoImpl) Delete(id *int64) (*m.RhcConnection, error) {
	var rhcConnection m.RhcConnection

	// Check if rhc connection exists for given tenant
	_, err := s.GetById(id)
	if err != nil {
		return nil, err
	}

	// The foreign key and the "cascade on delete" in the join table takes care of deleting the related
	// "source_rhc_connection" row.
	result := s.getDb().
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Delete(&rhcConnection)

	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_ids": rhcConnection.Sources, "rhc_connection_id": *id}).Errorf("Unable to delete RHC connection: %s", err)

		return nil, fmt.Errorf(`failed to delete rhcConnection with id "%d": %s`, id, result.Error)
	} else {
		logger.Log.WithFields(logrus.Fields{"tenant_id": *s.TenantID, "source_ids": rhcConnection.Sources, "rhc_connection_id": rhcConnection.ID}).Info("RHC connection deleted")

		return &rhcConnection, nil
	}
}

func (s *rhcConnectionDaoImpl) ListForSource(sourceId *int64, limit, offset int, filters []util.Filter) ([]m.RhcConnection, int64, error) {
	rhcConnections := make([]m.RhcConnection, 0)

	query := s.getDb().
		Model(&m.RhcConnection{}).
		Joins(`INNER JOIN "source_rhc_connections" "sr" ON "rhc_connections"."id" = "sr"."rhc_connection_id"`).
		Where(`"sr"."source_id" = ?`, sourceId).
		Where(`"sr"."tenant_id" = ?`, s.TenantID)

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	// Getting the total count (filters included) for pagination.
	count := int64(0)
	query.Count(&count)

	// Run the actual query.
	err = query.Limit(limit).Offset(offset).Find(&rhcConnections).Error
	if err != nil {
		return nil, 0, util.NewErrBadRequest(err)
	}

	return rhcConnections, count, nil
}
