package main

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getMetaDataDao func(c echo.Context) (dao.MetaDataDao, error)

func getMetaDataDaoWithoutTenant(_ echo.Context) (dao.MetaDataDao, error) {
	// we do not need tenancy for metadata.

	return &dao.MetaDataDaoImpl{}, nil
}

func MetaDataList(c echo.Context) error {
	applicationDB, err := getMetaDataDao(c)
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

	metaDatas, count, err := applicationDB.List(limit, offset, filters)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("Bad Request", "400"))
	}

	out := make([]interface{}, len(metaDatas))
	for i, a := range metaDatas {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Path(), int(*count), limit, offset))
}

func MetaDataGet(c echo.Context) error {
	metaDataDB, err := getMetaDataDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	c.Logger().Infof("Getting MetaData ID %v", id)

	app, err := metaDataDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}
