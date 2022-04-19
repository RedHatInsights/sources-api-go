package main

import (
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
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetRhcConnectionDao(&tenantId), nil
}

func RhcConnectionList(c echo.Context) error {
	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
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
		return util.NewErrBadRequest(err)
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	rhcConnection, err := rhcConnectionDao.GetById(&rhcConnectionId)
	if err != nil {
		return err
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
		return util.NewErrBadRequest(fmt.Sprintf("Validation failed: %s", err.Error()))
	}

	rhcConnection := &model.RhcConnection{
		RhcId:   input.RhcId,
		Extra:   input.Extra,
		Sources: []model.Source{{ID: input.SourceId}},
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	connection, err := rhcConnectionDao.Create(rhcConnection)
	if err != nil {
		return err
	}

	setEventStreamResource(c, connection)

	return c.JSON(http.StatusCreated, connection.ToResponse())
}

func RhcConnectionEdit(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	input := &model.RhcConnectionEditRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	dbRhcConnection, err := rhcConnectionDao.GetById(&rhcConnectionId)
	if err != nil {
		return err
	}

	dbRhcConnection.UpdateFromRequest(input)
	err = rhcConnectionDao.Update(dbRhcConnection)
	if err != nil {
		return err
	}

	setEventStreamResource(c, dbRhcConnection)

	return c.JSON(http.StatusOK, dbRhcConnection.ToResponse())
}

func RhcConnectionDelete(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	rhcConnection, err := rhcConnectionDao.Delete(&rhcConnectionId)
	if err != nil {
		return err
	}

	setEventStreamResource(c, rhcConnection)

	return c.NoContent(http.StatusNoContent)
}

// RhcConnectionSourcesList returns all the sources related to a given connection.
func RhcConnectionSourcesList(c echo.Context) error {
	paramId := c.Param("id")

	rhcConnectionId, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	// Check if the given rhcConnection exists.
	rhcConnectionDao, err := getRhcConnectionDao(c)
	if err != nil {
		return err
	}

	_, err = rhcConnectionDao.GetById(&rhcConnectionId)
	if err != nil {
		return err
	}

	sourceDao, err := getSourceDao(c)
	if err != nil {
		return err
	}

	// Get the list of sources for the given rhcConnection
	sources, count, err := sourceDao.ListForRhcConnection(&rhcConnectionId, limit, offset, filters)
	if err != nil {
		return err
	}

	out := make([]interface{}, len(sources))
	for i := 0; i < len(sources); i++ {
		out[i] = sources[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
