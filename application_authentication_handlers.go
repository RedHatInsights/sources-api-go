package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getApplicationAuthenticationDao func(c echo.Context) (dao.ApplicationAuthenticationDao, error)

func getApplicationAuthenticationDaoWithTenant(c echo.Context) (dao.ApplicationAuthenticationDao, error) {
	var tenantID int64
	var ok bool

	tenantVal := c.Get("tenantID")
	if tenantID, ok = tenantVal.(int64); !ok {
		return nil, fmt.Errorf("failed to pull tenant from request")
	}

	return &dao.ApplicationAuthenticationDaoImpl{TenantID: &tenantID}, nil
}

func ApplicationAuthenticationList(c echo.Context) error {
	applicationDB, err := getApplicationAuthenticationDao(c)
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

	applications, count, err := applicationDB.List(limit, offset, filters)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}
	c.Logger().Infof("tenant: %v", *applicationDB.Tenant())

	out := make([]interface{}, len(applications))
	for i, a := range applications {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationAuthenticationGet(c echo.Context) error {
	applicationDB, err := getApplicationAuthenticationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	c.Logger().Infof("Getting ApplicationAuthentication ID %v", id)

	app, err := applicationDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}
