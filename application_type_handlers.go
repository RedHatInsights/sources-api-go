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
var getApplicationTypeDao func(c echo.Context) (dao.ApplicationTypeDao, error)

func getApplicationTypeDaoWithTenant(c echo.Context) (dao.ApplicationTypeDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	if tenantId == 0 && err == nil {
		return dao.GetApplicationTypeDao(nil), nil
	} else {
		return dao.GetApplicationTypeDao(&tenantId), nil
	}
}

func SourceListApplicationTypes(c echo.Context) error {
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

	var (
		appTypes []m.ApplicationType
		count    int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	appTypes, count, err = applicationTypeDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)

	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(appTypes))
	for i := 0; i < len(appTypes); i++ {
		out[i] = appTypes[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
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

	var (
		appTypes []m.ApplicationType
		count    int64
	)

	appTypes, count, err = applicationTypeDB.List(limit, offset, filters)

	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(appTypes))
	for i := 0; i < len(appTypes); i++ {
		out[i] = appTypes[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationTypeGet(c echo.Context) error {
	applicationTypeDB, err := getApplicationTypeDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	appType, err := applicationTypeDB.GetById(&id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, appType.ToResponse())
}
