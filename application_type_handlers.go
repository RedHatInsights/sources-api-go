package main

import (
	"fmt"
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
	var tenantID int64
	var ok bool
	tenantVal := c.Get("tenantID")

	// if we set the tenant on this request - include it. otherwise do not.
	if tenantVal != nil {
		tenantVal := c.Get("tenantID")
		if tenantID, ok = tenantVal.(int64); !ok {
			return nil, fmt.Errorf("failed to pull tenant from request")
		}

		return &dao.ApplicationTypeDaoImpl{TenantID: &tenantID}, nil
	} else {
		return &dao.ApplicationTypeDaoImpl{}, nil
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
		apptypes []m.ApplicationType
		count    int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	apptypes, count, err = applicationTypeDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)

	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(apptypes))
	for i := 0; i < len(apptypes); i++ {
		out[i] = apptypes[i].ToResponse()
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
		apptypes []m.ApplicationType
		count    int64
	)

	apptypes, count, err = applicationTypeDB.List(limit, offset, filters)

	if err != nil {
		return err
	}

	// converting the objects to the interface type so the collection response can process it
	// allocating the length of our collection (so it doesn't have to resize)
	out := make([]interface{}, len(apptypes))
	for i := 0; i < len(apptypes); i++ {
		out[i] = apptypes[i].ToResponse()
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
		return util.NewErrBadRequest(err.Error())
	}

	appType, err := applicationTypeDB.GetById(&id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, appType.ToResponse())
}
