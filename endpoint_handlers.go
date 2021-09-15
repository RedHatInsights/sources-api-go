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
var getEndpointDao func(c echo.Context) (dao.EndpointDao, error)

func getEndpointDaoWithTenant(c echo.Context) (dao.EndpointDao, error) {
	var tenantID int64
	var ok bool

	tenantVal := c.Get("tenantID")
	if tenantID, ok = tenantVal.(int64); !ok {
		return nil, fmt.Errorf("failed to pull tenant from request")
	}

	return &dao.EndpointDaoImpl{TenantID: &tenantID}, nil
}

func SourceListEndpoint(c echo.Context) error {
	endpointDB, err := getEndpointDao(c)
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
		endpoints []m.Endpoint
		count     *int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return err
	}

	endpoints, count, err = endpointDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("Bad Request", "400"))
	}
	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i, a := range endpoints {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Path(), int(*count), limit, offset))
}

func EndpointList(c echo.Context) error {
	endpointDB, err := getEndpointDao(c)
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
		endpoints []m.Endpoint
		count     *int64
	)

	endpoints, count, err = endpointDB.List(limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("Bad Request", "400"))
	}
	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i, a := range endpoints {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Path(), int(count), limit, offset))
}

func EndpointGet(c echo.Context) error {
	endpointDB, err := getEndpointDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	c.Logger().Infof("Getting Endpoint ID %v", id)

	app, err := endpointDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}
