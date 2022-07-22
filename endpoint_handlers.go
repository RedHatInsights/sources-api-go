package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getEndpointDao func(c echo.Context) (dao.EndpointDao, error)

func getEndpointDaoWithTenant(c echo.Context) (dao.EndpointDao, error) {
	tenantId, err := util.GetTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetEndpointDao(&tenantId), nil
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
		return util.NewErrBadRequest(err)
	}

	endpoints, count, err = endpointDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)

	if err != nil {
		return err
	}

	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i := 0; i < len(endpoints); i++ {
		out[i] = endpoints[i].ToResponse()
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
		return err
	}

	c.Logger().Infof("tenant: %v", *endpointDB.Tenant())

	out := make([]interface{}, len(endpoints))
	for i := 0; i < len(endpoints); i++ {
		out[i] = endpoints[i].ToResponse()
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
		return util.NewErrBadRequest(err)
	}

	c.Logger().Infof("Getting Endpoint ID %v", id)

	app, err := endpointDB.GetById(&id)

	if err != nil {
		return err
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
		return util.NewErrBadRequest(fmt.Sprintf("Validation failed: %s", err))
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
		AvailabilityStatus:   input.AvailabilityStatus,
		SourceID:             input.SourceID,
	}

	err = endpointDao.Create(endpoint)
	if err != nil {
		return err
	}

	setEventStreamResource(c, endpoint)
	return c.JSON(http.StatusCreated, endpoint.ToResponse())
}

func EndpointEdit(c echo.Context) error {
	endpointDao, err := getEndpointDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	endpoint, err := endpointDao.GetById(&id)
	if err != nil {
		return err
	}

	// Store the previous status before updating the endpoint.
	previousStatus := endpoint.AvailabilityStatus

	// If "PausedAt" contains a date it means that the endpoint was paused back then.
	if endpoint.PausedAt != nil {
		input := &m.ResourceEditPausedRequest{}
		if err := c.Bind(input); err != nil {
			return util.NewErrBadRequest(err)
		}

		if err := endpoint.UpdateFromRequestPaused(input); err != nil {
			return util.NewErrBadRequest(err)
		}
	} else {
		input := &m.EndpointEditRequest{}
		if err := c.Bind(input); err != nil {
			return util.NewErrBadRequest(err)
		}

		endpoint.UpdateFromRequest(input)
	}

	err = endpointDao.Update(endpoint)
	if err != nil {
		return err
	}

	setNotificationForAvailabilityStatus(c, previousStatus, endpoint)
	setEventStreamResource(c, endpoint)

	return c.JSON(http.StatusOK, endpoint.ToResponse())
}

func EndpointDelete(c echo.Context) error {
	endpointDao, err := getEndpointDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	// Check if the endpoint exists before proceeding.
	endpointExists, err := endpointDao.Exists(id)
	if err != nil {
		return err
	}

	if !endpointExists {
		return util.NewErrNotFound("endpoint")
	}

	c.Logger().Infof("Deleting Endpoint Id %v", id)

	// Cascade delete the endpoint.
	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}
	err = service.DeleteCascade(endpointDao.Tenant(), nil, "Endpoint", id, forwardableHeaders)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func EndpointListAuthentications(c echo.Context) error {
	authDB, err := getAuthenticationDao(c)
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

	id, err := strconv.ParseInt(c.Param("endpoint_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auths, count, err := authDB.ListForEndpoint(id, limit, offset, filters)
	if err != nil {
		return err
	}

	tenantId := authDB.Tenant()
	out := make([]interface{}, len(auths))
	for i := 0; i < len(auths); i++ {
		// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
		err := marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, &auths[i])
		if err != nil {
			return err
		}

		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
