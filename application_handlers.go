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
var getApplicationDao func(c echo.Context) (dao.ApplicationDao, error)

func getApplicationDaoWithTenant(c echo.Context) (dao.ApplicationDao, error) {
	var tenantID int64
	var ok bool

	tenantVal := c.Get("tenantID")
	if tenantID, ok = tenantVal.(int64); !ok {
		return nil, fmt.Errorf("failed to pull tenant from request")
	}

	return &dao.ApplicationDaoImpl{TenantID: &tenantID}, nil
}

func ApplicationList(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
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
		applications []m.Application
		count        int64
	)

	applications, count, err = applicationDB.List(limit, offset, filters)
	if err != nil {
		return err
	}

	c.Logger().Infof("tenant: %v", *applicationDB.Tenant())

	out := make([]interface{}, len(applications))
	for i := 0; i < len(applications); i++ {
		out[i] = applications[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationGet(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	c.Logger().Infof("Getting Application ID %v", id)

	app, err := applicationDB.GetById(&id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}

func ApplicationCreate(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	input := &m.ApplicationCreateRequest{}
	if err = c.Bind(input); err != nil {
		return err
	}

	err = service.ValidateApplicationCreateRequest(input)
	if err != nil {
		return util.NewErrBadRequest(fmt.Sprintf("Validation failed: %s", err.Error()))
	}

	application := &m.Application{
		Extra:             input.Extra,
		ApplicationTypeID: input.ApplicationTypeID,
		SourceID:          input.SourceID,
	}

	err = applicationDB.Create(application)
	if err != nil {
		return err
	}

	setEventStreamResource(c, application)
	return c.JSON(http.StatusCreated, application.ToResponse())
}

func ApplicationEdit(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	input := &m.ApplicationEditRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	app, err := applicationDB.GetById(&id)
	if err != nil {
		return err
	}

	app.UpdateFromRequest(input)
	err = applicationDB.Update(app)
	if err != nil {
		return err
	}

	setEventStreamResource(c, app)
	return c.JSON(http.StatusOK, app.ToResponse())
}

func ApplicationDelete(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	c.Logger().Infof("Deleting Application Id %v", id)

	app, err := applicationDB.Delete(&id)
	if err != nil {
		return err
	}

	setEventStreamResource(c, app)
	return c.NoContent(http.StatusNoContent)
}

func ApplicationListAuthentications(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	appID, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	auths, count, err := authDao.ListForApplication(appID, 100, 0, nil)
	if err != nil {
		return err
	}

	out := make([]interface{}, count)
	for i := 0; i < int(count); i++ {
		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), 100, 0))
}

func SourceListApplications(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
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
		applications []m.Application
		count        int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err.Error())
	}

	applications, count, err = applicationDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)
	if err != nil {
		return err
	}

	c.Logger().Infof("tenant: %v", *applicationDB.Tenant())

	out := make([]interface{}, len(applications))
	for i := 0; i < len(applications); i++ {
		out[i] = applications[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
