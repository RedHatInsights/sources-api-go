package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var getRhcConnectionDao func() dao.RhcConnectionDao

func getDefaultRhcConnectionDao() dao.RhcConnectionDao {
	return &dao.RhcConnectionDaoImpl{}
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

	rhcConnectionDao := getRhcConnectionDao()
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

	rhcConnectionDao := getRhcConnectionDao()
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
		RhcId: input.RhcId,
		Extra: input.Extra,
		AvailabilityStatus: model.AvailabilityStatus{
			AvailabilityStatus: input.AvailabilityStatus,
		},
		Sources: []model.Source{{ID: sourceId}},
	}

	rhcConnectionDao := getRhcConnectionDao()
	connection, err := rhcConnectionDao.Create(rhcConnection)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, connection.ToResponseCreation())
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

	rhcConnectionDao := getRhcConnectionDao()
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

	return c.NoContent(http.StatusNoContent)
}

func RhcConnectionDelete(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("invalid id provided ", "400"))
	}

	rhcConnectionDao := getRhcConnectionDao()
	err = rhcConnectionDao.Delete(&rhcConnectionId)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
