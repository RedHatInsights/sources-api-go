package main

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
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

	createRequest := m.AuthenticationCreateRequest{}
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

	var extraDb datatypes.JSON

	extraDb, err = json.Marshal(createRequest.Extra)

	if err != nil {
		return util.NewErrBadRequest(`invalid JSON given in "extra" field`)
	}

	secret := &m.Authentication{
		Name:         createRequest.Name,
		AuthType:     createRequest.AuthType,
		Username:     createRequest.Username,
		Password:     createRequest.Password,
		ExtraDb:      extraDb,
		ResourceType: dao.SecretResourceType,
		ResourceID:   *requestParams.TenantID,
	}
	err = secretDao.Create(secret)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	return c.JSON(http.StatusCreated, secret.SecretToResponse())
}
