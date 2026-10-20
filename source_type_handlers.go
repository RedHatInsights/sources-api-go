package main

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getSourceTypeDao func(c echo.Context) (dao.SourceTypeDao, error)

func getSourceTypeDaoWithoutTenant(_ echo.Context) (dao.SourceTypeDao, error) {
	// we do not need tenancy for source type.
	return dao.GetSourceTypeDao(), nil
}

func SourceTypeList(c echo.Context) error {
	sourceTypeDB, err := getSourceTypeDao(c)
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

	sourceTypes, count, err := sourceTypeDB.List(limit, offset, filters)
	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(sourceTypes))
	for i := 0; i < len(sourceTypes); i++ {
		out[i] = sourceTypes[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func SourceTypeGet(c echo.Context) error {
	SourceTypeDB, err := getSourceTypeDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	sourceType, err := SourceTypeDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, sourceType.ToResponse())
}
