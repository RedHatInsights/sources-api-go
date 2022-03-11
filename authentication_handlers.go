package main

import (
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var getAuthenticationDao func(c echo.Context) (dao.AuthenticationDao, error)

func getAuthenticationDaoWithTenant(c echo.Context) (dao.AuthenticationDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetAuthenticationDao(&tenantId), nil
}

func AuthenticationList(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
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

	authentications, count, err := authDao.List(limit, offset, filters)
	if err != nil {
		return err
	}

	out := make([]interface{}, 0, len(authentications))
	for _, auth := range authentications {
		out = append(out, *auth.ToResponse())
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func AuthenticationGet(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	auth, err := authDao.GetById(c.Param("uid"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, auth.ToResponse())
}

func AuthenticationCreate(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	createRequest := m.AuthenticationCreateRequest{}
	err = c.Bind(&createRequest)
	if err != nil {
		return err
	}

	err = service.ValidateAuthenticationCreationRequest(&createRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auth := &m.Authentication{
		Name:         createRequest.Name,
		AuthType:     createRequest.AuthType,
		Username:     createRequest.Username,
		Password:     createRequest.Password,
		Extra:        createRequest.Extra,
		ResourceType: createRequest.ResourceType,
		ResourceID:   createRequest.ResourceID,
	}
	err = authDao.Create(auth)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	setEventStreamResource(c, auth)
	return c.JSON(http.StatusCreated, auth.ToResponse())
}

func AuthenticationUpdate(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	updateRequest := &m.AuthenticationEditRequest{}
	err = c.Bind(updateRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auth, err := authDao.GetById(c.Param("uid"))
	if err != nil {
		return err
	}

	auth.UpdateFromRequest(updateRequest)
	err = authDao.Update(auth)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	setEventStreamResource(c, auth)
	return c.JSON(http.StatusOK, auth.ToResponse())
}

func AuthenticationDelete(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	auth, err := authDao.Delete(c.Param("uid"))
	if err != nil {
		return err
	}

	setEventStreamResource(c, auth)
	return c.NoContent(http.StatusNoContent)
}
