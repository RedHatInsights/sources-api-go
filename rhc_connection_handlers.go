package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var getRhcConnectionDao func(c echo.Context) (dao.RhcConnectionDao, error)

func getDefaultRhcConnectionDao(c echo.Context) (dao.RhcConnectionDao, error) {
	var tenantID int64
	var ok bool

	tenantVal := c.Get("tenantID")
	if tenantID, ok = tenantVal.(int64); !ok {
		return nil, fmt.Errorf("failed to pull tenant from request")
	}

	return &dao.RhcConnectionDaoImpl{TenantID: tenantID}, nil
}

func RhcConnectionList(c echo.Context) error {
	filters, err := getFilters(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	rhcConnections, count, err := rhcConnectionDao.List(limit, offset, filters)
	if err != nil {
		return err
	}

	out := make([]interface{}, len(rhcConnections))
	for i := 0; i < len(rhcConnections); i++ {
		out[i] = rhcConnections[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func RhcConnectionGetById(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid id provided ", "400"))
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	rhcConnection, err := rhcConnectionDao.GetById(&rhcConnectionId)

	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	return c.JSON(http.StatusOK, rhcConnection.ToResponse())
}

func RhcConnectionCreate(c echo.Context) error {
	input := &model.RhcConnectionCreateRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	err := service.ValidateRhcConnectionRequest(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	sourceId, err := strconv.ParseInt(input.SourceId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid source id", "400"))
	}

	rhcConnection := &model.RhcConnection{
		RhcId:   input.RhcId,
		Extra:   input.Extra,
		Sources: []model.Source{{ID: sourceId}},
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	connection, err := rhcConnectionDao.Create(rhcConnection)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, connection.ToResponse())
}

func RhcConnectionUpdate(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid id provided ", "400"))
	}

	input := &model.RhcConnectionUpdateRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	dbRhcConnection, err := rhcConnectionDao.GetById(&rhcConnectionId)
	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	dbRhcConnection.UpdateFromRequest(input)
	err = rhcConnectionDao.Update(dbRhcConnection)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dbRhcConnection.ToResponse())
}

func RhcConnectionDelete(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid id provided ", "400"))
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	err = rhcConnectionDao.Delete(&rhcConnectionId)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// RhcConnectionSourcesList returns all the sources related to a given connection.
func RhcConnectionSourcesList(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid id provided ", "400"))
	}

	filters, err := getFilters(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	// Check if the given rhcConnection exists.
	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	_, err = rhcConnectionDao.GetById(&rhcConnectionId)
	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	// Get the list of sources for the given rhcConnection
	sources, count, err := rhcConnectionDao.GetRelatedSourcesToId(&rhcConnectionId, limit, offset, filters)
	if err != nil {
		return err
	}

	out := make([]interface{}, len(sources))
	for i := 0; i < len(sources); i++ {
		out[i] = sources[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
