package main

import (
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getApplicationAuthenticationDao func(c echo.Context) (dao.ApplicationAuthenticationDao, error)

func getApplicationAuthenticationDaoWithTenant(c echo.Context) (dao.ApplicationAuthenticationDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetApplicationAuthenticationDao(&tenantId), nil
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
		return err
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
		return util.NewErrBadRequest(err)
	}

	c.Logger().Infof("Getting ApplicationAuthentication ID %v", id)

	app, err := applicationDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}

func ApplicationAuthenticationListAuthentications(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("application_authentication_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	auths, count, err := authDao.ListForApplicationAuthentication(id, 100, 0, nil)
	if err != nil {
		return c.JSON(http.StatusNotFound, util.ErrorDoc(err.Error(), "404"))
	}

	out := make([]interface{}, count)
	for i := 0; i < int(count); i++ {
		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), 100, 0))
}
