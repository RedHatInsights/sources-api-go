package main

import (
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
var getApplicationAuthenticationDao func(c echo.Context) (dao.ApplicationAuthenticationDao, error)

func getApplicationAuthenticationDaoWithTenant(c echo.Context) (dao.ApplicationAuthenticationDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetApplicationAuthenticationDao(&tenantId), nil
}

func ApplicationAuthenticationList(c echo.Context) error {
	appAuthDB, err := getApplicationAuthenticationDao(c)
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

	appAuths, count, err := appAuthDB.List(limit, offset, filters)
	if err != nil {
		return err
	}
	c.Logger().Infof("tenant: %v", *appAuthDB.Tenant())

	out := make([]interface{}, len(appAuths))
	for i, a := range appAuths {
		out[i] = *a.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationAuthenticationGet(c echo.Context) error {
	appAuthDB, err := getApplicationAuthenticationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	c.Logger().Infof("Getting ApplicationAuthentication ID %v", id)

	appAuth, err := appAuthDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, appAuth.ToResponse())
}

func ApplicationAuthenticationCreate(c echo.Context) error {
	appAuthDB, err := getApplicationAuthenticationDao(c)
	if err != nil {
		return err
	}

	input := m.ApplicationAuthenticationCreateRequest{}
	if err := c.Bind(&input); err != nil {
		return err
	}

	err = service.ValidateApplicationAuthenticationCreateRequest(&input)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	appAuth := &m.ApplicationAuthentication{
		ApplicationID:    input.ApplicationID,
		AuthenticationID: input.AuthenticationID,
	}

	err = appAuthDB.Create(appAuth)
	if err != nil {
		return err
	}

	accountNumber, err := getAccountNumberFromEchoContext(c)
	if err != nil {
		c.Logger().Warn(err)
	}

	appAuth.Tenant = m.Tenant{Id: appAuth.TenantID, ExternalTenant: accountNumber}
	setEventStreamResource(c, appAuth)

	return c.JSON(http.StatusCreated, appAuth.ToResponse())
}

func ApplicationAuthenticationDelete(c echo.Context) error {
	appAuthDB, err := getApplicationAuthenticationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	appAuth, err := appAuthDB.Delete(&id)
	if err != nil {
		return err
	}

	setEventStreamResource(c, appAuth)

	return c.NoContent(http.StatusNoContent)
}

func ApplicationAuthenticationListAuthentications(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("application_authentication_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auths, count, err := authDao.ListForApplicationAuthentication(id, 100, 0, nil)
	if err != nil {
		return err
	}

	tenantId := authDao.Tenant()
	out := make([]interface{}, count)
	for i := 0; i < int(count); i++ {
		// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
		err := marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, &auths[i])
		if err != nil {
			return err
		}

		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), 100, 0))
}
