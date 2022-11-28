package main

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var getSecretDao func(c echo.Context) (dao.SecretDao, error)

func getSecretDaoWithTenant(c echo.Context) (dao.SecretDao, error) {
	requestParams, err := dao.NewRequestParamsFromContext(c)
	if err != nil {
		return nil, err
	}

	return dao.GetSecretDao(requestParams), nil
}

func SecretCreate(c echo.Context) error {
	secretDao, err := getSecretDao(c)
	if err != nil {
		return err
	}

	createRequest := m.SecretCreateRequest{}
	err = c.Bind(&createRequest)
	if err != nil {
		return err
	}

	requestParams, err := dao.NewRequestParamsFromContext(c)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	err = service.ValidateSecretCreationRequest(requestParams, createRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	secret := &m.Authentication{
		Name:     createRequest.Name,
		AuthType: createRequest.AuthType,
		Username: createRequest.Username,
	}

	err = secret.SetExtra(createRequest.Extra)
	if err != nil {
		return err
	}

	err = secret.SetPassword(createRequest.Password)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	if createRequest.UserScoped && requestParams.UserID != nil {
		secret.UserID = requestParams.UserID
	}

	err = secretDao.Create(secret)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	return c.JSON(http.StatusCreated, secret.ToSecretResponse())
}

func SecretList(c echo.Context) error {
	secretDao, err := getSecretDao(c)
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

	secrets, count, err := secretDao.List(limit, offset, filters)
	if err != nil {
		span := trace.SpanFromContext(c.Request().Context())
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	out := make([]interface{}, 0, len(secrets))
	for _, secret := range secrets {
		out = append(out, *secret.ToSecretResponse())
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func SecretGet(c echo.Context) error {
	secretDao, err := getSecretDao(c)
	if err != nil {
		return err
	}

	paramID, err := util.InterfaceToInt64(c.Param("id"))
	if err != nil {
		span := trace.SpanFromContext(c.Request().Context())
		span.SetStatus(codes.Error, err.Error())

		return util.NewErrBadRequest(err)
	}

	secret, err := secretDao.GetById(&paramID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, secret.ToSecretResponse())
}

func SecretEdit(c echo.Context) error {
	secretDao, err := getSecretDao(c)
	if err != nil {
		return err
	}

	updateRequest := &m.SecretEditRequest{}
	err = c.Bind(updateRequest)
	if err != nil {
		return err
	}

	paramID, err := util.InterfaceToInt64(c.Param("id"))
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	secret, err := secretDao.GetById(&paramID)
	if err != nil {
		return err
	}

	err = secret.UpdateSecretFromRequest(updateRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	err = secretDao.Update(secret)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, secret.ToSecretResponse())
}

func SecretDelete(c echo.Context) error {
	secretDao, err := getSecretDao(c)
	if err != nil {
		return err
	}

	secretID, err := util.InterfaceToInt64(c.Param("id"))
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	err = secretDao.Delete(&secretID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
