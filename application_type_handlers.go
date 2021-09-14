package main

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getApplicationTypeDao func(c echo.Context) (dao.ApplicationTypeDao, error)

func getApplicationTypeDaoWithoutTenant(_ echo.Context) (dao.ApplicationTypeDao, error) {
	// we do not need tenancy for application type.
	return &dao.ApplicationTypeDaoImpl{}, nil
}

func ApplicationTypeList(c echo.Context) error {
	applicationTypeDB, err := getApplicationTypeDao(c)
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

	apptypes, count, err := applicationTypeDB.List(limit, offset, filters)
	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(apptypes))
	for i, s := range apptypes {
		out[i] = *s.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Path(), int(count), limit, offset))
}

func ApplicationTypeGet(c echo.Context) error {
	applicationTypeDB, err := getApplicationTypeDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	appType, err := applicationTypeDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, appType.ToResponse())
}
