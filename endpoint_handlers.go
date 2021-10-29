package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
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
		count     int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	endpoints, count, err = endpointDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}
	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i, a := range endpoints {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
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
		count     int64
	)

	endpoints, count, err = endpointDB.List(limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}
	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i, a := range endpoints {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func EndpointGet(c echo.Context) error {
	endpointDB, err := getEndpointDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	c.Logger().Infof("Getting Endpoint ID %v", id)

	app, err := endpointDB.GetById(&id)
	if err != nil {
		return c.JSON(http.StatusNotFound, util.ErrorDoc(err.Error(), "404"))
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}

func EndpointCreate(c echo.Context) error {
	endpointDao, err := getEndpointDao(c)
	if err != nil {
		return err
	}

	input := &m.EndpointCreateRequest{}
	err = c.Bind(input)
	if err != nil {
		return err
	}

	err = service.ValidateEndpointCreateRequest(endpointDao, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(fmt.Sprintf("Validation failed: %s", err.Error()), "400"))
	}

	endpoint := &m.Endpoint{
		Default:              &input.Default,
		ReceptorNode:         input.ReceptorNode,
		Role:                 &input.Role,
		Scheme:               input.Scheme,
		Host:                 &input.Host,
		Port:                 input.Port,
		Path:                 &input.Path,
		VerifySsl:            input.VerifySsl,
		CertificateAuthority: input.CertificateAuthority,
		AvailabilityStatus:   m.AvailabilityStatus{AvailabilityStatus: input.AvailabilityStatus},
		SourceID:             input.SourceID,
	}

	err = endpointDao.Create(endpoint)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, endpoint.ToResponse())
}
