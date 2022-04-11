package main

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getMetaDataDao func(c echo.Context) (dao.MetaDataDao, error)

func getMetaDataDaoWithoutTenant(c echo.Context) (dao.MetaDataDao, error) {
	return dao.GetMetaDataDao(), nil
}

func MetaDataList(c echo.Context) error {
	metaDataDB, err := getMetaDataDao(c)
	if err != nil {
		return err
	}

	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	var (
		metaDatas []m.MetaData
		count     int64
	)

	metaDatas, count, err = metaDataDB.List(limit, offset, filters)

	if err != nil {
		return err
	}

	out := make([]interface{}, len(metaDatas))
	for i := 0; i < len(metaDatas); i++ {
		out[i] = metaDatas[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationTypeListMetaData(c echo.Context) error {
	metaDataDB, err := getMetaDataDao(c)
	if err != nil {
		return err
	}

	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("application_type_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	var (
		metaDatas []m.MetaData
		count     int64
	)

	metaDatas, count, err = metaDataDB.ListForApplicationType(&id, limit, offset, filters)

	if err != nil {
		return err
	}

	out := make([]interface{}, len(metaDatas))
	for i := 0; i < len(metaDatas); i++ {
		out[i] = metaDatas[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func MetaDataGet(c echo.Context) error {
	metaDataDB, err := getMetaDataDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	c.Logger().Infof("Getting MetaData ID %v", id)

	app, err := metaDataDB.GetById(&id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}
